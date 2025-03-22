package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
)

// 재귀적으로 URL 디코딩
func recursiveURLDecode(s string) string {
	prev := ""
	for s != prev {
		prev = s
		decoded, err := url.QueryUnescape(s)
		if err != nil {
			break
		}
		s = decoded
	}
	return s
}

// 퓨니코드 디코딩
func decodePunycode(s string) string {
	// 도메인만 디코딩 (스킴 제거 후, 호스트만 추출해서 디코딩 시도)
	if !strings.Contains(s, "xn--") {
		return s
	}
	// 스킴 제거하고 도메인만 바꿔치기
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	decodedHost, err := idna.ToUnicode(u.Host)
	if err != nil {
		return s
	}
	u.Host = decodedHost
	return u.String()
}

// 요청 라인에서 전체 URL 추출
func extractURL(line string) (fullMatch string, method string, rawURL string) {
	re := regexp.MustCompile(`"(GET|POST|PUT|DELETE|HEAD|OPTIONS|PATCH) ([^ ]+) HTTP/`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 3 {
		return matches[0], matches[1], matches[2]
	}
	return "", "", ""
}

func main() {
	file, err := os.Open("access.log")
	if err != nil {
		fmt.Println("파일 열기 오류:", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		fullMatch, method, rawURL := extractURL(line)
		if rawURL == "" {
			continue
		}

		// 재귀적으로 URL 디코딩
		urlDecoded := recursiveURLDecode(rawURL)

		// 퓨니코드 디코딩
		finalDecoded := decodePunycode(urlDecoded)

		// 원래 라인의 URL 부분을 디코딩된 URL로 치환
		newRequestLine := fmt.Sprintf("\"%s %s HTTP/", method, finalDecoded)
		newLine := strings.Replace(line, fullMatch, newRequestLine, 1)

		fmt.Println(newLine)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("파일 읽기 오류:", err)
	}
}
