package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const BYTES = 2048
const NAME = "goGet"

func main() {

	continued := false
	flag.Usage = usage

	// Set the flag to the variable
	flag.BoolVar(&continued, "continue", false, NAME+" --continue [url]")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf("Input URL is missing.\n")
		os.Exit(1)
	}
	url_reader(args[0], continued)
}

func usage() {

	fmt.Fprintf(os.Stderr, "Usage: %s [url]\n", NAME)

	flag.PrintDefaults()
	os.Exit(2)
}

// Function to get the size of the local file, used when the downloading is continued
func file_size(file_name string) int64 {

	file, err := os.Open(file_name)
	if err != nil {
		return 0
	}
	defer file.Close()

	file_stat, err := file.Stat()
	return file_stat.Size()
}

// Showing the status of the download in console
func show_data(url string, file_name string, remote_size int64, file_part int64, percent *float64) {

	new_percent := math.Floor((float64(file_part) / float64(remote_size)) * 100.00)
	// Only refresh the data with integers
	if new_percent > *percent {
		clear := exec.Command("clear")
		clear.Stdout = os.Stdout
		clear.Run()
		fmt.Printf("Downloading file from %s\n", url)
		fmt.Printf("Saving file to: %s\n", file_name)
		fmt.Printf("%.0f%% of 100%%\n", new_percent)
		fmt.Printf("%d of %d bytes\n", file_part, remote_size)
	}
	*percent = new_percent
}

// Function for retrieve the data of the file from URL
func get_url(url string, continued bool) (string, int64, int64, *http.Response, error) {

	remote_size := int64(0)
	file_part := int64(0)
	file_name := string("")

	resp, err := http.Get(url)
	if err != nil {
		return file_name, remote_size, file_part, resp, err
	}
	last_slash := strings.LastIndex(resp.Request.URL.Path, "/") // Getting the index of the last slash
	file_name = resp.Request.URL.Path[last_slash+1:] // Saving the name of the file

	if continued {
		file_part = file_size(file_name) // Getting size of the local file

		// Saving the size of the remote file
		if file_part == resp.ContentLength {
			remote_size = resp.ContentLength
		// If local file and remote has diferent size, then set the range to download
		} else if file_part != resp.ContentLength {
			range_part := fmt.Sprintf("bytes=%d-%d", file_part, resp.ContentLength)
			resp.Request.Header.Add("Range", range_part)
			// Doing a new request with the set range
			client := &http.Client{}
			resp, err = client.Do(resp.Request)
			if err != nil {
				return file_name, remote_size, file_part, resp, err
			}
			remote_size = resp.ContentLength + file_part
		}
	} else {
		os.Remove(file_name) // If there is a previous file, this is deleted
		file, err := os.OpenFile(file_name, os.O_RDWR|os.O_CREATE, 0666) // Create a new file
		if err != nil {
			return file_name, remote_size, file_part, resp, err
		}
		file.Close()
		remote_size = resp.ContentLength
	}
	return file_name, remote_size, file_part, resp, err
}

// Function to get the file from URL and save it in to disc
func url_reader(url string, continued bool) int64 {

	file_name, remote_size, file_part, resp, err := get_url(url, continued)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	accum := int64(file_part) // Read bytes
	percent := float64(0)
	buffer := make([]byte, BYTES)

	// Loop to read the remote file and write the local file
	for {
		n, err := resp.Body.Read(buffer)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		if n == 0 {
			return accum
		}
		if remote_size == accum {
			fmt.Printf("File is already on disc\n")
			return accum
		}
		err = file_writer(file_name, buffer[0:n], accum)
		if err != nil {
			log.Fatal(err)
		}
		accum += int64(n)
		// Refresh data in console
		show_data(url, file_name, remote_size, accum, &percent)
	}
	return accum
}

// Function to write the buffer in to the local file
func file_writer(file_name string, buffer []byte, offset int64) error {
	// For read and write access, if file doesn't exist this is created.
	file, err := os.OpenFile(file_name, os.O_RDWR|os.O_CREATE, 0660)
	if err != nil {
		return err
	}
	defer file.Close()
	// The write begin with an offset
	_, err = file.WriteAt(buffer, offset)
	if err != nil {
		return err
	}
	return nil
}
