package commandline

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lexesjan/go-web-proxy-server/pkg/log"
	"github.com/mgutz/ansi"
)

// Prompt is the commandline prompt which has colours
var Prompt = fmt.Sprintf(
	"%s[%s%sProxy%s%s]$%s ",
	ansi.LightGreen,
	ansi.Reset,
	ansi.ColorCode("white+bh"),
	ansi.Reset,
	ansi.LightGreen,
	ansi.Reset,
)

// Dispatcher handles the user input and modifies the blockList
func Dispatcher(blockList *sync.Map) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(Prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			log.ProxyError(err)
		}

		trimmedInput := strings.TrimRight(input, "\n")
		tokens := strings.Split(trimmedInput, " ")
		command := tokens[0]
		if command != "" {
			switch command {
			case "block":
				if len(tokens) == 1 {
					fmt.Fprintf(os.Stderr, "usage: block <domain name>\n")
					continue
				}

				website := tokens[1]
				_, loaded := blockList.LoadOrStore(website, true)
				if !loaded {
					fmt.Printf("%s: blocked %q\n", command, website)
				} else {
					fmt.Fprintf(os.Stderr, "%s: website %q already blocked\n", command, website)
				}
			case "unblock":
				if len(tokens) == 1 {
					fmt.Fprintf(os.Stderr, "usage: unblock <domain name>\n")
				}

				website := tokens[1]
				_, found := blockList.LoadAndDelete(website)
				if found {
					fmt.Printf("%s: unblocked %q\n", command, website)
				} else {
					fmt.Fprintf(os.Stderr, "%s: website %q not blocked\n", command, website)
				}
			default:
				fmt.Printf("proxy: %q: command not found\n", tokens[0])
			}
		}
	}
}
