package steward

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/fsnotify/fsnotify"
)

// getMessagesFromFile will start a file watcher for the given directory
// and filename. It will take a channel of []byte as input, and it is
// in this channel the content of a file that has changed is returned.
func getMessagesFromFile(directoryToCheck string, fileName string, fileContentCh chan []byte) {
	fileUpdated := make(chan bool)
	go fileWatcherStart(directoryToCheck, fileUpdated)

	for {
		select {
		case <-fileUpdated:
			//load file, read it's content
			b, err := readTruncateMessageFile(fileName)
			if err != nil {
				log.Printf("error: reading file: %v", err)
			}

			fileContentCh <- b
			fmt.Printf("File content read: %s\n", b)
		}
	}

}

// readTruncateMessageFile, will read all the messages in the given
// file, and truncate the file after read.
// A []byte will be returned with the content read.
func readTruncateMessageFile(fileName string) ([]byte, error) {

	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_RDWR, os.ModeAppend)
	if err != nil {
		log.Printf("Failed to open file %v\n", err)
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	lines := []byte{}

	for scanner.Scan() {
		lines = append(lines, scanner.Bytes()...)
	}

	fmt.Printf("*** DEBUG : %s\n", lines)

	fmt.Printf("read: %s\n", lines)

	// empty the file after all is read
	ret, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("f.Seek failed: %v", err)
	}
	fmt.Printf("** ret=%v\n", ret)

	err = f.Truncate(0)
	if err != nil {
		fmt.Printf("******* %#v\n", err)
		return nil, fmt.Errorf("f.Truncate failed: %v", err)
	}

	return lines, nil
}

func fileWatcherStart(directoryToCheck string, fileUpdated chan bool) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Failed fsnotify.NewWatcher")
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		//Give a true value to updated so it reads the file the first time.
		fileUpdated <- true
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					//testing with an update chan to get updates
					fileUpdated <- true
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(directoryToCheck)
	if err != nil {
		log.Printf("error: watcher add: %v\n", err)
	}
	<-done
}