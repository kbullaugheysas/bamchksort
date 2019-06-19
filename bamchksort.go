package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

//func strnum_cmp(a, b string) int {
//	return -1
//}

func digitToInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		log.Fatal("failed to parse digit ", err)
	}
	return n
}

// From: https://github.com/samtools/samtools/blob/develop/bam_sort.c#L13
func strnum_cmp(as, bs string) int {
	a := []rune(as)
	b := []rune(bs)
	i := 0
	j := 0
	for i < len(a) && j < len(b) {
		if unicode.IsDigit(a[i]) && unicode.IsDigit(b[j]) {
			for i < len(a) && a[i] == '0' {
				i++
			}
			for j < len(b) && b[j] == '0' {
				j++
			}
			for i < len(a) && j < len(b) && unicode.IsDigit(a[i]) && unicode.IsDigit(b[j]) && a[i] == b[j] {
				i++
				j++
			}
			// By this point we've forwarded across any leading zeros && any digits that match.
			// Next we get determine if they have the same number of digits
			// before the first non-diget. If so we use the numerical values of
			// the number formed by these digits to determine order.
			if i < len(a) && j < len(b) && unicode.IsDigit(a[i]) && unicode.IsDigit(b[j]) {
				k := 0
				for i+k < len(a) && unicode.IsDigit(a[i+k]) && j+k < len(b) && unicode.IsDigit(b[j+k]) {
					k += 1
				}
				if i+k < len(a) && unicode.IsDigit(a[i+k]) {
					return 1
				} else if j+k < len(b) && unicode.IsDigit(b[j+k]) {
					return -1
				} else {
					return digitToInt(string(a[i:(i+k)])) - digitToInt(string(b[j:(j+k)]))
				}
			} else if i < len(a) && unicode.IsDigit(a[i]) {
				return 1
			} else if j < len(b) && unicode.IsDigit(b[j]) {
				return -1
			} else if i != j {
				if i < j {
					return 1
				}
				return -1
			}
		} else {
			if a[i] != b[j] {
				return digitToInt(string(a[i])) - digitToInt(string(b[j]))
			}
			i++
			j++
		}
	}
	if len(a) > len(b) {
		return 1
	} else if len(a) < len(b) {
		return -1
	}
	return 0
}

func usage() {
	fmt.Println("usage: bamchksort bamfile")
	fmt.Println("   or: bamchksort -")
}

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(1)
	}
	bam_fn := os.Args[1]
	var scanner *bufio.Scanner
	var cmd *exec.Cmd
	if bam_fn == "-" {
		scanner = bufio.NewScanner(os.Stdin)
	} else {
		cmd = exec.Command("samtools", "view", bam_fn)
		bam, err := cmd.StdoutPipe()
		if err != nil {
			log.Fatal("failed creating pipe: ", err)
		}
		if err := cmd.Start(); err != nil {
			log.Fatal("command failed to start: ", err)
		}
		scanner = bufio.NewScanner(bam)
	}
	var wg sync.WaitGroup
	var prev string
	var line_no int
	wg.Add(1)
	go func() {
		defer wg.Done()
		for scanner.Scan() {
			line_no += 1
			line := scanner.Text()
			line = strings.TrimSpace(line)
			fields := strings.Split(line, "\t")
			read_name := fields[0]
			if prev != "" {
				if strnum_cmp(prev, read_name) > 0 {
					log.Fatalf("sorting order violated at line %d", line_no)
				}
			}
			prev = read_name
		}
		if err := scanner.Err(); err != nil {
			log.Fatal("scanner errored: ", err)
		}
	}()

	wg.Wait()

	if bam_fn != "-" {
		if err := cmd.Wait(); err != nil {
			log.Fatal("wait failed: ", err)
		}
	}

	fmt.Println("OK", line_no)
}
