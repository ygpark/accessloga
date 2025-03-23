package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/idna"
)

// 디코딩 옵션 구조체
type decodeOptions struct {
	OnlyURL      bool
	OnlyPunycode bool
	OnlyBase64   bool
}

// --- 디코딩 유틸 함수들 ---

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

func decodePunycode(s string) string {
	if !strings.Contains(s, "xn--") {
		return s
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	decodedHost, err := idna.ToUnicode(u.Host)
	if err != nil {
		return s
	}
	u.Host = decodedHost
	return rebuildURL(u)
}

func tryBase64Decode(s string) string {
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(s)
		if err != nil {
			return s
		}
	}
	return string(decoded)
}

func decodeWreplyQueryParam(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := u.Query()
	if val, ok := query["wreply"]; ok {
		for i := range val {
			decodedVal, err := url.QueryUnescape(val[i])
			if err != nil {
				decodedVal = val[i]
			}
			query["wreply"][i] = tryBase64Decode(decodedVal)
		}
		u.RawQuery = query.Encode()
	}
	return rebuildURL(u)
}

func decodeWreplyInLine(line string) string {
	re := regexp.MustCompile(`wreply=([^&\s"]+)`)
	return re.ReplaceAllStringFunc(line, func(match string) string {
		parts := strings.SplitN(match, "=", 2)
		if len(parts) != 2 {
			return match
		}
		decodedVal, err := url.QueryUnescape(parts[1])
		if err != nil {
			decodedVal = parts[1]
		}
		decoded := tryBase64Decode(decodedVal)
		return fmt.Sprintf("wreply=%s", decoded)
	})
}

func decodeAllURLEncodedParts(line string, opts decodeOptions) string {
	re := regexp.MustCompile(`(?:https?|ftp)%3A%2F%2F[^\s"]+|%[0-9A-Fa-f]{2}`)
	return re.ReplaceAllStringFunc(line, func(encoded string) string {
		decoded := recursiveURLDecode(encoded)
		if opts.OnlyPunycode {
			decoded = decodePunycode(decoded)
		} else if !opts.OnlyURL {
			decoded = decodePunycode(decoded)
		}
		return decoded
	})
}

func decodeAllPunycodeDomains(line string, opts decodeOptions) string {
	re := regexp.MustCompile(`https?://([a-zA-Z0-9.-]*xn--[a-zA-Z0-9-]+(?:\.[a-zA-Z0-9.-]+)*)`)
	return re.ReplaceAllStringFunc(line, func(match string) string {
		u, err := url.Parse(match)
		if err != nil {
			return match
		}
		decodedHost, err := idna.ToUnicode(u.Host)
		if err != nil {
			return match
		}
		u.Host = decodedHost
		return rebuildURL(u)
	})
}

func rebuildURL(u *url.URL) string {
	if u.Host == "" {
		return u.RequestURI()
	}
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, u.RequestURI())
}

func decodeURL(raw string, opts decodeOptions) string {
	if opts.OnlyBase64 {
		return decodeWreplyQueryParam(raw)
	}
	if opts.OnlyURL {
		return recursiveURLDecode(raw)
	}
	if opts.OnlyPunycode {
		return decodePunycode(raw)
	}
	raw = recursiveURLDecode(raw)
	raw = decodeWreplyQueryParam(raw)
	raw = decodePunycode(raw)
	return raw
}

func decodeLine(line string, opts decodeOptions) string {
	fullMatch, method, rawURL := extractURL(line)
	if rawURL != "" {
		decoded := decodeURL(rawURL, opts)
		newRequestLine := fmt.Sprintf("\"%s %s HTTP/", method, decoded)
		line = strings.Replace(line, fullMatch, newRequestLine, 1)
	}

	line = decodeWreplyInLine(line)
	if !opts.OnlyBase64 {
		line = decodeAllURLEncodedParts(line, opts)
		line = decodeAllPunycodeDomains(line, opts)
	}

	return line
}

func extractURL(line string) (fullMatch, method, rawURL string) {
	re := regexp.MustCompile(`"(GET|POST|PUT|DELETE|HEAD|OPTIONS|PATCH) ([^ ]+) HTTP/`)
	matches := re.FindStringSubmatch(line)
	if len(matches) >= 3 {
		return matches[0], matches[1], matches[2]
	}
	return "", "", ""
}

func validateFlags(urlOnly, punyOnly, base64Only bool) {
	if (urlOnly && punyOnly) || (urlOnly && base64Only) || (punyOnly && base64Only) {
		fmt.Fprintln(os.Stderr, "옵션 충돌: -decode-only-url, -decode-only-punycode, -decode-only-base64 는 동시에 사용할 수 없습니다.")
		os.Exit(1)
	}
}

func processLines(scanner *bufio.Scanner, opts decodeOptions, writer *bufio.Writer) {
	for scanner.Scan() {
		line := scanner.Text()
		decodedLine := decodeLine(line, opts)
		writer.WriteString(decodedLine + "\n")
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("입력 읽기 오류:", err)
	}
	writer.Flush()
}

const version = "v1.0.0"

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `accessloga - access.log URL/Base64/Punycode 디코더

사용법:
  accessloga [옵션] [파일명]
  cat access.log | accessloga [옵션]

옵션:
  -version                  버전 정보 출력
  -decode-only-url         URL 디코딩만 수행
  -decode-only-punycode    퓨니코드 디코딩만 수행
  -decode-only-base64      wreply 파라미터 base64 디코딩만 수행
  -o <파일명>               출력 파일 경로 (기본값: 표준 출력)
  -h, --help               도움말 표시

`)
	}
	decodeURLOnly := flag.Bool("decode-only-url", false, "URL 디코딩만 수행")
	decodePunycodeOnly := flag.Bool("decode-only-punycode", false, "퓨니코드 디코딩만 수행")
	decodeBase64Only := flag.Bool("decode-only-base64", false, "wreply 파라미터 base64 디코딩만 수행")
	outputFile := flag.String("o", "", "출력 파일 경로 (없으면 콘솔 출력)")
	versionFlag := flag.Bool("version", false, "버전 정보 출력")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("accessloga %s", version)
		return
	}

	validateFlags(*decodeURLOnly, *decodePunycodeOnly, *decodeBase64Only)

	opts := decodeOptions{
		OnlyURL:      *decodeURLOnly,
		OnlyPunycode: *decodePunycodeOnly,
		OnlyBase64:   *decodeBase64Only,
	}

	var output *os.File
	var err error
	if *outputFile != "" && *outputFile != "-" && *outputFile != "stdout" {
		output, err = os.Create(*outputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "출력 파일 생성 실패: %v\n", err)
			os.Exit(1)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}
	writer := bufio.NewWriter(output)

	if flag.NArg() > 0 {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "파일 열기 실패: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		processLines(scanner, opts, writer)
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		processLines(scanner, opts, writer)
	}
}
