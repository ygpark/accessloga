# accessloga

👉 Apache/Nginx access.log 파일에서 URL, base64, 푸니코드 문자열을 자동으로 디코딩해주는 CLI 도구입니다.

---

## ✨ 기능

- URL 퍼센트 인코딩 디코딩 (`%2F` → `/`)
- 푸니코드 도메인 디코딩 (`xn--` → 한국 도메인)
- `wreply` 파라미터의 base64 디코딩 (`dGVzdEBuY29t` → `test@com`)
- 로그 파일, 파이프 입력 모드 지원
- 선택적 옵션 처리 지원

---

## 파일 설치

```bash
go install github.com/ygpark/accessloga@latest
```

또는 로컬 빌드:

```bash
git clone https://github.com/yourname/accessloga.git
cd accessloga
go build -o accessloga
```

---

## 사용법

### 기본 사용

```bash
accessloga access.log
```

### 파이프 사용

```bash
cat access.log | accessloga
```

### 결과를 파일로 저장

```bash
accessloga -o result.log access.log
```

---

## 옵션

| 옵션                     | 설명                                |
| ------------------------ | ----------------------------------- |
| `--decode-only-url`      | URL 디코딩만 수행                   |
| `--decode-only-punycode` | 푸니코드 디코딩만 수행              |
| `--decode-only-base64`   | `wreply` 파라미터만 base64 디코딩   |
| `-o <파일>`              | 출력 파일 지정 (default: 컨설 출력) |
| `--version`              | 버전 정보 출력                      |
| `-h`, `--help`           | 도움말 표시                         |

> ⚠️ `--decode-only-*` 옵션은 동시에 사용할 수 없습니다.

---

## 예시

```log
입력:
"GET /login?wreply=dGVzdEBuY29t HTTP/1.1"

출력:
"GET /login?wreply=test@com HTTP/1.1"
```

---

## 라이센스

MIT License

---
