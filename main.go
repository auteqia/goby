package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

func usage() {
	fmt.Println(`usage : 
	-h --help
	-t : number of threads
	-u : target url  
	-w : path to wordlist  
	-q Quiet mode, disabled by default  
	-p Port target  
	--recursive for subdirectory  
	-h for help  
	--redirect for 301 follow (default)  
	`)
}

type flags struct {
	TargetUrl    string
	DictFile     string
	Worker       int
	Quietmode    bool
	Redirect     bool
	Recursive    bool
	Help         bool
	RedColor     string
	Nonecolor    string
	BlueColor    string
	MovingColor  string
	Width        int
	MaxDepth     int
	CurrentDepth int
}

func scanURL(urls chan string, subDirectories chan string, subDirUrls chan string, wg *sync.WaitGroup, subDirWg *sync.WaitGroup, mutex *sync.Mutex, sentURLs map[string]struct{}, opts flags) {
	defer wg.Done()
	var client *http.Client
	//http.client for avoiding new tls handshake at every request for better perf
	if opts.Redirect {
		client = &http.Client{}
	} else {
		//don't follow 301 redirect
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	for {
		select {
		//if the channel is not empty
		case urlToScan := <-urls:
			response, err := client.Get(opts.TargetUrl + urlToScan)
			if err != nil {
				fmt.Printf("Error making GET request: %v\n", err)
				return
			}
			defer response.Body.Close()

			//enter if quiet mode and not recursive
			if opts.Quietmode && !opts.Recursive {
				//if quiet mode
				if response.StatusCode == 200 {

					//%-*s format with dynamic width
					fmt.Printf("%-*s: %s[%d]%s \n", opts.Width, urlToScan, opts.BlueColor, response.StatusCode, opts.Nonecolor)

				}
				continue
			}
			//enter if recursive and directory
			if opts.Recursive && isDirectory(opts.TargetUrl+urlToScan) {
				fmt.Printf("%-*s: %s[directory]%s \n", opts.Width, urlToScan, opts.BlueColor, opts.Nonecolor)
				subdirectoryURL := urlToScan
				subDirectories <- subdirectoryURL
				// fmt.Println("sending", subdirectoryURL)
				select {
				case <-subDirectories:
					//adds one worker for each subdirectory encountered
					subDirWg.Add(1)
					subDirectories <- urlToScan
					go scanSubdirectory(subDirectories, subDirUrls, subDirWg, mutex, sentURLs, opts)
				default:
					continue
				}

			} else { // default case
				switch {
				case response.StatusCode == 301:
					opts.MovingColor = "\033[38;5;82m"
				case response.StatusCode == 200:
					opts.MovingColor = "\033[38;5;45m"
				case response.StatusCode == 404:
					opts.MovingColor = "\033[38;5;196m"
				}
				if opts.Quietmode {
					if response.StatusCode == 200 {
						fmt.Printf("%-*s: %s[%d]%s \n", opts.Width, urlToScan, "\033[38;5;45m", response.StatusCode, opts.Nonecolor)
					}
					continue
				}
				//print [directory] instead of 301 when it's a directory
				if response.StatusCode == 301 && isDirectory(opts.TargetUrl+urlToScan) {
					fmt.Printf("%-*s: %s[directory]%s \n", opts.Width, urlToScan, "\033[38;5;45m", opts.Nonecolor)

				} else {
					fmt.Printf("%-*s: %s[%d]%s \n", opts.Width, urlToScan, opts.MovingColor, response.StatusCode, opts.Nonecolor)
				}
			}
		default:
			return
		}
	}
}

// urlToScan is avered subdirectory
func scanSubdirectory(subDirectories chan string, subDirUrls chan string, subDirWg *sync.WaitGroup, mutex *sync.Mutex, sentURLs map[string]struct{}, opts flags) {
	defer subDirWg.Done()

	client := &http.Client{}
	for {
		select {
		//Avered subdirectories
		case urlToScan, more := <-subDirectories:
			if !more {
				return
			}

			wordlist, err := readDictionary(opts.DictFile)
			if err != nil {
				logrus.Fatal(err)
			}

			// Process wordlist for the current subdirectory
			for _, word := range wordlist {
				fullURL := opts.TargetUrl + urlToScan + "/" + word
				//cache system
				mutex.Lock()
				if _, exists := sentURLs[fullURL]; exists {
					mutex.Unlock()
					continue
				}
				//cache system
				sentURLs[fullURL] = struct{}{}
				mutex.Unlock()

				response, err := client.Get(fullURL)
				if err != nil {
					fmt.Printf("Error making GET request: %v\n", err)
					os.Exit(1)
				}

				// Determine color based on status code
				switch {
				case response.StatusCode == 301:
					opts.MovingColor = "\033[38;5;82m"
				case response.StatusCode == 200:
					opts.MovingColor = "\033[38;5;45m"
				case response.StatusCode == 404:
					opts.MovingColor = "\033[38;5;196m"
				default:
					opts.MovingColor = "\033[38;5;214m"
				}

				//quiet mode when recursive
				if opts.Quietmode {
					if response.StatusCode == 200 {
						fmt.Printf("%-*s: %s%s[%d]%s\n", opts.Width, urlToScan+"/"+word, opts.Nonecolor, opts.MovingColor, response.StatusCode, opts.Nonecolor)
					}
				} else {
					fmt.Printf("%-*s: %s%s[%d]%s\n", opts.Width, urlToScan+"/"+word, opts.Nonecolor, opts.MovingColor, response.StatusCode, opts.Nonecolor)
				}

				// Check if it's a directory and send it to subDirectories
				if isDirectory(fullURL) && opts.CurrentDepth < opts.MaxDepth {
					fmt.Printf("%-*s: %s[directory]%s\n", opts.Width, urlToScan+"/", opts.BlueColor, opts.Nonecolor)

					mutex.Lock()
					sentURLs[fullURL] = struct{}{}
					mutex.Unlock()

					subDirectories <- urlToScan + word
					//for max-depth scanning
					opts.CurrentDepth++
				}
			}
		default:
			return
		}
	}
}

func parseArgs() flags {
	opts := flags{}
	flag.StringVar(&opts.TargetUrl, "t", "", "Target URL")
	//get a string
	flag.StringVar(&opts.DictFile, "d", "", "Path to wordlist")
	flag.IntVar(&opts.Worker, "w", 1, "Number of workers")
	flag.BoolVar(&opts.Quietmode, "q", false, "Enable quiet mode")
	flag.BoolVar(&opts.Redirect, "redirect", false, "Enable follow redirect mode for 301 HTTP code")
	flag.BoolVar(&opts.Recursive, "recursive", false, "Enable recursive mode")
	flag.IntVar(&opts.MaxDepth, "max-depth", 1, "Depth in recursive mode")
	flag.BoolVar(&opts.Help, "h", false, "Show usage")
	flag.Parse()

	if opts.Help {
		usage()
		os.Exit(1)
	}

	if opts.TargetUrl == "" || opts.DictFile == "" {
		fmt.Println("Not enough args, exiting \n ")
		usage()
		os.Exit(1)
	}
	return opts
}

func constructSubdirectoryURL(baseURL, subdirectory string) string {
	if strings.HasSuffix(baseURL, "/") {
		return baseURL + subdirectory
	}
	return baseURL + "/" + subdirectory
}

func getAbolutePath(file string) string {
	ex, err := filepath.Abs(file)
	if err != nil {
		panic(err)
	}
	return ex
}

func banner(opts flags) {

	asciiart := `


                                                                     
        _____              ____     ______  ______   ______   _____  
   _____\    \_        ____\_  \__  \     \|\     \ |\     \ |     | 
  /     /|     |      /     /     \  |     |\|     |\ \     \|     | 
 /     / /____/|     /     /\      | |     |/____ /  \ \           | 
|     | |_____|/    |     |  |     | |     |\     \   \ \____      | 
|     | |_________  |     |  |     | |     | |     |   \|___/     /| 
|\     \|\        \ |     | /     /| |     | |     |       /     / | 
| \_____\|    |\__/||\     \_____/ |/_____/|/_____/|      /_____/  / 
| |     /____/| | ||| \_____\   | / |    |||     | |      |     | /  
 \|_____|     |\|_|/ \ |    |___|/  |____|/|_____|/       |_____|/   
        |____/        \|____|                                        

`
	fmt.Println(asciiart)
	width := 20
	fmt.Println("------------------------------")
	//%-*s format with dynamic width
	fmt.Printf("%-*s: %s%s%s\n", width, "METHOD ", "\033[38;5;45m", "GET", "\033[0m")
	fmt.Printf("%-*s: %s\n", width, "Target URL ", opts.TargetUrl)
	fmt.Printf("%-*s: %s\n", width, "Wordlist ", getAbolutePath(opts.DictFile))
	fmt.Printf("%-*s: %s\n", width, "Response handled ", "200, 204, 301, 302, 401, 404")
	fmt.Printf("%-*s: %s%d%s\n", width, "Worker(s) ", "\033[38;5;45m", opts.Worker, "\033[0m")
	fmt.Println("------------------------------ \n ")

}

func isDirectory(fullURL string) bool {
	// Enable 301 redirect recognition. Override the default choice
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	// fmt.Println("heyy")

	// Check without the / if I get 301 moved permanently
	//I could check if there's directory listing with /test/ but it's not allowed everywhere and should be disabled ^^
	response, err := client.Get(fullURL)
	if err != nil {
		fmt.Printf("Error making GET request: %v\n", err)
		return false
	}
	defer response.Body.Close()

	if response.StatusCode == 301 || response.StatusCode == http.StatusFound {
		// Follow the redirect
		redirectedURL := response.Header.Get("Location")
		if redirectedURL == "" {
			return false
		}

		// Check if the redirected URL ends with a slash
		if strings.HasSuffix(redirectedURL, "/") {
			// Directory found
			return true
		}
	}

	// Default false
	return false
}

func stripURL(url string) string {
	urlStripped := strings.Trim(url, "FUZZ")
	return urlStripped
}

func readDictionary(PathOfFile string) ([]string, error) {
	file, err := os.Open(PathOfFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var words []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		word := scanner.Text()
		//add each word scanned
		words = append(words, word)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	//return slice of string
	return words, nil
}

func main() {
	opts := parseArgs()
	banner(opts)
	opts.Width = 50
	opts.Nonecolor = "\033[0m"
	opts.BlueColor = "\033[38;5;45m"
	opts.TargetUrl = stripURL(opts.TargetUrl)
	opts.CurrentDepth = 1
	var mutex sync.Mutex

	// Create a WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup
	var subDirWg sync.WaitGroup

	sentURLs := make(map[string]struct{})

	//read a wordlist file and returns words in slice of string
	wordlist, err := readDictionary(opts.DictFile)
	if err != nil {
		logrus.Fatal(err)
	}
	// -1 because I start from 0 in the loop
	numberOfWords := (len(wordlist))

	//check if there are more workers than words in the dic
	if opts.Worker > numberOfWords {
		fmt.Println("More worker than words, reducing number of worker at number ", numberOfWords)
		opts.Worker = numberOfWords
	}
	// The subDirectories is a channel that is used to communicate between the scanURL and scanSubdirectory functions.
	subDirectories := make(chan string, len(wordlist))

	urls := make(chan string, len(wordlist))
	//create a chan from the words read in wordlist

	//for each words read from wordlist in subdir func
	subDirUrls := make(chan string, len(wordlist))

	start := time.Now()

	for i := 0; i < opts.Worker; i++ {
		//add worker iteration
		wg.Add(1)
		//be careful this is not a DOS program ^^
		go scanURL(urls, subDirectories, subDirUrls, &wg, &subDirWg, &mutex, sentURLs, opts)

	}
	for _, word := range wordlist {
		//this will send the result of a job in a queue
		urls <- word
	}
	wg.Wait()
	subDirWg.Wait()
	close(urls)
	close(subDirUrls)
	close(subDirectories)
	// Wait for all subdirectory workers to finish
	fmt.Println("Time elapsed : ", time.Since(start))
}
