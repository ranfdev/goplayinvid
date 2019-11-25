package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"mime"
)
const VERSION = 0.1

type VideoFormat struct {
	Url     string
	Type    string
	Quality string
}
type VideoItem struct {
	Title           string
	VideoId         string
	Description     string
	Genre           string
	Author          string
	LengthSeconds   int32
	Rating          float32
	ViewCount       int64
	LikeCount       int64
	AdaptiveFormats []VideoFormat
}

func (v *VideoItem) printDetails() {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "Title:\t%s\n", v.Title)
	fmt.Fprintf(w, "Id:\t%s\n", v.VideoId)
	fmt.Fprintf(w, "Length (s):\t%d\n", v.LengthSeconds)
	fmt.Fprintf(w, "Views:\t%d\n", v.ViewCount)
	fmt.Fprintf(w, "Likes:\t%d\n", v.LikeCount)
	fmt.Fprintf(w, "Rating:\t%f\n", v.Rating)
	w.Flush()
	fmt.Printf("Description:\n%s\n", v.Description)
}

func printVideos(videos []VideoItem) {
	fmt.Println()
	fmt.Printf("  N | TITLE\n")
	for i := 0; i < len(videos); i++ {
		fmt.Printf("%3d %v\n", i, videos[i].Title)
	}
	fmt.Println()
}
func playFromId(cmd string, videoId string) {
	v := getVideoById(videoId)
	// Try to play each format. If it plays it successfully, we are done (break) else,
	// try to play the next format
	fmt.Println("Playing", v.Title)
	for _, format := range v.AdaptiveFormats {
		// Filter format by type. We only need audio streams
		if strings.Contains(format.Type, "audio") {
			cmd := exec.Command(cmd, format.Url)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				log.Print(err)
			} else {
				break
			}
		}
	}
}
func parseRange(input string) (start, end int, err error) {
	var err1, err2 error

	strrange := strings.Split(input, "-")
	start, err1 = strconv.Atoi(strrange[0])

	if err1 != nil {
		return 0, 0, errors.New("Invalid start of range")
	}

	if len(strrange) > 1 {
		end, err2 = strconv.Atoi(strrange[1])
		if err2 != nil {
			return 0, 0, errors.New("Invalid end of range")
		}
	} else {
		end = start
	}

	return start, end, nil
}
func getVideoById(videoId string) VideoItem {
	var v VideoItem
	// download the video info (i need to do this to get a list of AdaptiveFormats)
	resp, _ := http.Get("https://invidio.us/api/v1/videos/" + videoId)
	json.NewDecoder(resp.Body).Decode(&v)
	return v
}
func printHelp() {
	fmt.Println("Goplayinvid")
	fmt.Println("Version", VERSION)
	fmt.Println("<n> to play the song <n>")
	fmt.Println("<n>-<m> to play the song from <n> to <m>")
	fmt.Println("2-5,8 to play the songs 2,3,4,5,8")
	fmt.Println("i<n> to see the info of the song <n>")
	fmt.Println("d<n> to download the song <n> using youtube-dl")
}
func downloadFromId(videoId string) {
}
func main() {
	var line string
	var videos []VideoItem
	var queue []VideoItem

	playerCmd := os.Getenv("GOINVID_PLAYER_CMD")
	if playerCmd == "" {
		log.Println("Using default player: mpv")
		testMpv := exec.Command("mpv", "-V")
		err := testMpv.Run()
		if err != nil {
			log.Fatal("Cannot find mpv player")
		}
		playerCmd = "mpv"
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		// show prompt
		if len(videos) > 0 {
			fmt.Print("Type a number 'n' or range 'n-m' to play songs. ")
		}
		fmt.Print("Type 'h' for help. To search a song, type '/songname'. ")
		fmt.Println("'q' to exit")
		fmt.Print("> ")
		scanner.Scan()
		line = scanner.Text()
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		// handle input
		switch line[0] {
		case 'q':
			return
		case 'h':
			printHelp()
		case 'i':
			if len(videos) == 0 {
				fmt.Println("Before using this function, you have to search something with /name")
				continue
			}
			i, err := strconv.Atoi(string(line[1]))
			if err != nil {
				log.Fatal("Failed to parse n after i")
			}
			v := getVideoById(videos[i].VideoId)
			v.printDetails()
			fmt.Println("Press enter to continue")
			scanner.Scan()
		case 'd':
			if len(videos) == 0 {
				fmt.Println("Before using this function, you have to search something with /name")
				continue
			}
			i, err := strconv.Atoi(string(line[1]))
			if err != nil {
				log.Fatal(err)
			}

			v := getVideoById(videos[i].VideoId)
			fmt.Println("Available formats:")
			for x, f := range v.AdaptiveFormats {
				fmt.Println(x, f.Type, f.Quality)
			}
			scanner.Scan()
			j, err := strconv.Atoi(scanner.Text())
			if err != nil {
				log.Fatal(err)
			}

			mime.AddExtensionType(".webm", "audio/webm")
			mime.AddExtensionType(".webm", "video/webm")
			mime.AddExtensionType(".mp4", "audio/mp4")
			mime.AddExtensionType(".mp4", "video/mp4")

			downloader := "curl"
			filename := v.Title
			mimeType := v.AdaptiveFormats[j].Type
			fmt.Println(mimeType)
			ext, err := mime.ExtensionsByType(mimeType)
			if err != nil {
				log.Println(err)
				continue
			}
			filename += ext[0]
				log.Println("Using downloader", downloader)
				cmd := exec.Command(downloader, v.AdaptiveFormats[j].Url, "-L", "-o", filename)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				err = cmd.Run()
				if err != nil {
					log.Println("Error running", downloader)
				} else {
					break
				}
		case '/':
			q := url.Values{}
			// line[1:] to remove the first character ("/")
			q.Set("q", line[1:])
			resp, _ := http.Get("https://invidio.us/api/v1/search?" + q.Encode())
			json.NewDecoder(resp.Body).Decode(&videos)
		default:
			if len(videos) == 0 {
				fmt.Println("Before using this function, you have to search something with /name")
				continue
			}
			// if the input is neither "q", neither "/..." then it probably is a range
			// of numbers (e.g. 2-5).
			// You can have multiple ranges, separated by a ',' (2-5,7-9)
			ranges := strings.Split(line, ",")
			for r := 0; r < len(ranges); r++ {
				start, end, err := parseRange(ranges[r])
				if err != nil {
					log.Print(err)
					continue
				}
				if start < 0 || end >= len(videos) {
					log.Print("Range not valid")
					continue
				}
				// play songs in range
				for i := start; i <= end; i++ {
					queue = append(queue, videos[i])
				}

			}
			fmt.Println("Queue")
			printVideos(queue)
			fmt.Println("Playing Queue. To exit from the queue, you have to close the program with Ctrl-C")
			for len(queue) > 0 {
				v := queue[0]
				playFromId(playerCmd, v.VideoId)
				queue = queue[1:]
			}
		}
		printVideos(videos)
	}
}
