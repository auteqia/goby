# Installation

- Download a pre-built binary from releases page, unpack and run :p

- `git clone https://github.com/strange-fruit/goby.git ; cd goby ; go get ; go build`

- Clone the projet and run it with `go run main.go` with the flags needed

Note : goby@v1.0.0 requires go >= 1.21.5;

# Usage

```

-d 
	string - Path for a dictionnary file

-w
	int - Number of workers

-t 
	string - Target URL

-q 
	bool - Enabe quiet mode (only HTTP 200 printed)

--recursive 
	bool - Scan subdirectories if found.

--max-depth
	int - Subdirectory depth (default 1)

--redirect
	bool - Follow redirect and prints HTTP 200 instead of 301

-h
	bool - Display usage

```

  
  
# Example usage

#### This section is based with the `git clone` install without compilation

### Basic fuzz

```bash
go run main.go -t http://localhost/FUZZ -d /usr/share/seclists/Discovery/Web-Content/common.txt 
```


### Concurrency fuzzing

```bash
go run main.go -t http://localhost/FUZZ -d /usr/share/seclists/Discovery/Web-Content/common.txt -w 6
```

`-w` is the number of goroutines spawned

### Don't follow 301 Redirect
```bash
go run main.go -t http://localhost/FUZZ -d /usr/share/seclists/Discovery/Web-Content/common.txt --redirect
```

By default, goby follow HTTP Response code 301 and send a 200 OK. With --redirect, goby displays if this is a 301 instead of following the redirect.

### Recursive scanning

```bash
go run main.go -t http://localhost/FUZZ -d /usr/share/seclists/Discovery/Web-Content/common.txt --recursive
```

When `--recursive` is set, it also scans for subdirectories when found.

#### Max-depth scanning
```bash
go run main.go -t http://localhost/FUZZ -d /usr/share/seclists/Discovery/Web-Content/common.txt --recursive --max-depth 2
```
By default depth is set to 1 with `--recursive`. It allows deep scanning with subdirectory found. 

When `--timeout` is set, each HTTP GET is sent with a delay. The value is in millisecond.

### Complete Command
```bash
go run main.go -t http://localhost/FUZZ -d /usr/share/seclists/Discovery/Web-Content/common.txt --recursive --max-depth 2 -q -w 7
```


# Licence

See <a href="https://github.com/strange-fruit/goby/blob/master/LICENCE"> MIT Licence </a>

## Docs related

https://pkg.go.dev/net/http

https://www.digitalocean.com/community/tutorials/how-to-make-http-requests-in-go

https://pkg.go.dev/flag

https://pkg.go.dev/net/http

https://www.golang-book.com/books/intro/10

https://zetcode.com/golang/flag/

https://github.com/ffuf/ffuf/blob/master/pkg/runner/simple.go

https://stackoverflow.com/questions/23297520/how-can-i-make-the-go-http-client-not-follow-redirects-automatically
