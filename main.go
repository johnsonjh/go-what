///////////////////////////////////////////////////////////////////////////////////////////////////
// go-what - main.go
// Copyright (c) 2016 MIT PDOS
// Copyright (c) 2025 Jeffrey H. Johnson
// SPDX-License-Identifier: MIT
// scspell-id: 0520e114-83c9-11f0-b56a-80ee73e9b8e7
///////////////////////////////////////////////////////////////////////////////////////////////////

// go-what
package main

// This is a Golang conversion of https://github.com/mit-pdos/what

// `go-what` is an improved version of the `w` tool.  It finds all processes associated with a
// TTY (not just those registered in `wtmp`), and reports all users that are running anything.
// In particular, unlike `w`, `go-what` will also show things running in detached screens/tmuxen.

///////////////////////////////////////////////////////////////////////////////////////////////////

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

///////////////////////////////////////////////////////////////////////////////////////////////////

type TTY struct {
	Name      string
	Stat      syscall.Stat_t
	Processes []string
}

///////////////////////////////////////////////////////////////////////////////////////////////////

func prettyTime(ts int64) string {
	diff := time.Now().Unix() - ts
	days := diff / (24 * 60 * 60)
	diff %= 24 * 60 * 60
	hours := diff / (60 * 60)
	diff %= 60 * 60
	mins := diff / 60
	secs := diff % 60

	if days > 99 {
		return fmt.Sprintf("%5dd",
			days)
	}

	if days > 0 {
		return fmt.Sprintf("%2dd%02dh",
			days, hours)
	}

	if hours > 0 {
		return fmt.Sprintf("%2dh%02dm",
			hours, mins)
	}

	if mins > 0 {
		return fmt.Sprintf("%2dm%02ds",
			mins, secs)
	}

	return fmt.Sprintf("%5ds",
		secs)
}

///////////////////////////////////////////////////////////////////////////////////////////////////

func getTermSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80, 24
	}

	return w, h
}

///////////////////////////////////////////////////////////////////////////////////////////////////

func main() {
	ttys := make(map[uint64]*TTY)
	ttyGlobs := []string{"/dev/tty*", "/dev/pts/*"}

	for _, glob := range ttyGlobs {
		files, _ := filepath.Glob(glob)
		for _, file := range files {
			var stat syscall.Stat_t

			err := syscall.Stat(file, &stat)
			if err != nil {
				continue
			}

			ttys[stat.Rdev] = &TTY{Name: file[5:], Stat: stat}
		}
	}

	notty := make(map[uint32]int)
	uids := make(map[uint32]bool)

	procFiles, _ := os.ReadDir("/proc")
	for _, f := range procFiles {
		pid, err := strconv.Atoi(f.Name())
		if err != nil {
			continue
		}

		statPath := fmt.Sprintf("/proc/%d/stat",
			pid)

		statContent, err := os.ReadFile(statPath) //nolint:gosec
		if err != nil {
			continue
		}

		cmdlinePath := fmt.Sprintf("/proc/%d/cmdline",
			pid)

		cmdlineContent, err := os.ReadFile(cmdlinePath) //nolint:gosec
		if err != nil {
			continue
		}

		var procStat syscall.Stat_t

		err = syscall.Stat(fmt.Sprintf("/proc/%d",
			pid),
			&procStat)
		if err != nil {
			continue
		}

		uids[procStat.Uid] = true

		i := strings.LastIndex(string(statContent), ")")
		if i == -1 {
			continue
		}

		parts := strings.Fields(string(statContent)[i+2:])

		ttyNr, _ := strconv.ParseUint(parts[4], 10, 64)
		tpgid, _ := strconv.Atoi(parts[5])

		if ttyNr == 0 || tpgid == -1 {
			notty[procStat.Uid]++

			continue
		}

		cmdline := string(cmdlineContent)
		if strings.HasPrefix(cmdline, "/sbin/getty") ||
			strings.HasPrefix(cmdline, "/sbin/agetty") ||
			strings.HasPrefix(cmdline, "tmux") ||
			strings.HasPrefix(cmdline, "screen") ||
			strings.HasPrefix(cmdline, "dtach") ||
			strings.HasPrefix(cmdline, "-zsh") ||
			strings.HasPrefix(cmdline, "-ksh") ||
			strings.HasPrefix(cmdline, "-ksh93") ||
			strings.HasPrefix(cmdline, "-sh") ||
			strings.HasPrefix(cmdline, "-bash") ||
			strings.HasPrefix(cmdline, "/sbin/mingetty") {
			continue
		}

		tty, ok := ttys[ttyNr]
		if ok && tpgid == pid {
			tty.Processes = append(tty.Processes, strings.ReplaceAll(cmdline, "\x00", " "))
		}
	}

	sortedTtys := make([]*TTY, 0, len(ttys))

	for _, tty := range ttys {
		sortedTtys = append(sortedTtys, tty)
	}

	sort.Slice(sortedTtys, func(i, j int) bool {
		return sortedTtys[i].Stat.Atim.Sec < sortedTtys[j].Stat.Atim.Sec
	})

	uptimeContent, _ := os.ReadFile("/proc/uptime")
	uptimeParts := strings.Split(string(uptimeContent), " ")
	uptime, _ := strconv.ParseFloat(uptimeParts[0], 64)

	loadavgContent, _ := os.ReadFile("/proc/loadavg")
	loadavgParts := strings.Split(string(loadavgContent), " ")

	fmt.Printf(" up %s  %2d users  load %s %s %s  procs %s\n",
		strings.TrimSpace(prettyTime(time.Now().Unix()-int64(uptime))), len(uids),
		loadavgParts[0], loadavgParts[1], loadavgParts[2], loadavgParts[3])

	cols, _ := getTermSize()

	fmt.Printf("% -8s %-7s %6s %6s %6s %s\n",
		"USER", "TTY", "LOGIN", "\x1b[4mINPUT\x1b[0m", "OUTPUT", "WHAT")

	uidColors := make(map[uint32]int)
	colors := []int{32, 33, 35, 36}

	loggedInUids := make(map[uint32]bool)

	for _, tty := range sortedTtys {
		if len(tty.Processes) > 0 {
			loggedInUids[tty.Stat.Uid] = true
		}
	}

	for _, tty := range sortedTtys {
		if len(tty.Processes) == 0 {
			continue
		}

		if _, ok := uidColors[tty.Stat.Uid]; !ok {
			uidColors[tty.Stat.Uid] = len(uidColors) % len(colors)
		}

		color := fmt.Sprintf("\x1b[%dm",
			colors[uidColors[tty.Stat.Uid]])

		u, err := user.LookupId(strconv.Itoa(int(tty.Stat.Uid)))
		if err != nil || u == nil {
			u = &user.User{Username: strconv.Itoa(int(tty.Stat.Uid))}
		}

		for _, cmd := range tty.Processes {
			line := fmt.Sprintf("% -8.8s %-7s %6s %6s %6s %s",
				u.Username, tty.Name, prettyTime(tty.Stat.Ctim.Sec),
				prettyTime(tty.Stat.Atim.Sec), prettyTime(tty.Stat.Mtim.Sec), cmd)
			if len(line) > cols {
				line = line[:cols]
			}

			fmt.Println(color + line + "\x1b[0m")
		}
	}

	if _, ok := notty[0]; !ok {
		notty[0] = 0
	}

	var nottyUids []uint32

	for uid := range notty {
		_, ok := loggedInUids[uid]
		if ok || uid == 0 {
			nottyUids = append(nottyUids, uid)
		}
	}

	slices.Sort(nottyUids)

	for _, uid := range nottyUids {
		count := notty[uid]

		u, err := user.LookupId(strconv.Itoa(int(uid)))
		if err != nil || u == nil {
			u = &user.User{Username: strconv.Itoa(int(uid))}
		}

		fmt.Printf("% -8.8s %-7s %d more processes\n",
			u.Username, "none", count)
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////
