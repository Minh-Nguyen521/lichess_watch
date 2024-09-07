package main

import (
	"bufio"
	"encoding/json"
	"image/color"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

var (
	squareSize float32 = 50
	lightColor         = color.RGBA{R: 240, G: 240, B: 240, A: 255}
	darkColor          = color.RGBA{R: 96, G: 96, B: 96, A: 255}
)

type Data struct {
	FEN string `json:"fen"`
	LM  string `json:"lm"`
	WC  int    `json:"wc"`
	BC  int    `json:"bc"`
}

type FeedResponse struct {
	T string `json:"t"`
	D Data   `json:"d"`
}

func ConvertFENToArray(fen string) [8][8]string {
	var board [8][8]string
	if fen == "" {
		return board
	}

	rows := strings.Split(fen, " ")[0]
	fenRows := strings.Split(rows, "/")
	for i, row := range fenRows {
		col := 0
		for _, char := range row {
			if char >= '1' && char <= '8' {
				numEmpty := int(char - '0')
				for j := 0; j < numEmpty; j++ {
					board[i][col] = "."
					col++
				}
			} else {
				board[i][col] = string(char)
				col++
			}
		}
	}
	return board
}

func fetchData(fenChan chan<- string) {
	defer close(fenChan)

	req, err := http.NewRequest("GET", "http://lichess.org/api/tv/best/feed", nil)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Accept", "application/x-ndjson")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed with status: %s", resp.Status)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		var data FeedResponse
		err := json.Unmarshal([]byte(line), &data)
		if err != nil {
			log.Printf("Error unmarshalling JSON: %v", err)
			continue
		}
		log.Println("Received FEN:", data.D.FEN)
		fenChan <- data.D.FEN
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func update(fenChan <-chan string, myWindow fyne.Window) {
	for NewFen := range fenChan {
		if NewFen == "" {
			NewFen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
		}

		if !isValidFEN(NewFen) {
			log.Printf("Invalid FEN string: %s", NewFen)
			NewFen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
		}

		chessboard := container.NewGridWithColumns(8)

		addPiece := func(imagePath string, square *canvas.Rectangle) fyne.CanvasObject {
			absPath, err := filepath.Abs(imagePath)
			if err != nil {
				return square
			}
			if _, err := os.Stat(absPath); err == nil {
				piece := canvas.NewImageFromFile(absPath)
				piece.FillMode = canvas.ImageFillContain
				piece.SetMinSize(fyne.NewSize(squareSize, squareSize))
				return container.NewStack(square, piece)
			}
			return square
		}

		board := ConvertFENToArray(NewFen)

		for row := 0; row < 8; row++ {
			for col := 0; col < 8; col++ {
				var squareColor color.Color
				if (row+col)%2 == 0 {
					squareColor = lightColor
				} else {
					squareColor = darkColor
				}

				square := canvas.NewRectangle(squareColor)
				square.SetMinSize(fyne.NewSize(squareSize, squareSize))

				var piecePath string
				switch board[row][col] {
				case "r":
					piecePath = "image/br.png"
				case "n":
					piecePath = "image/bn.png"
				case "b":
					piecePath = "image/bb.png"
				case "q":
					piecePath = "image/bq.png"
				case "k":
					piecePath = "image/bk.png"
				case "p":
					piecePath = "image/bp.png"
				case "P":
					piecePath = "image/wp.png"
				case "R":
					piecePath = "image/wr.png"
				case "N":
					piecePath = "image/wn.png"
				case "B":
					piecePath = "image/wb.png"
				case "Q":
					piecePath = "image/wq.png"
				case "K":
					piecePath = "image/wk.png"
				}

				chessboard.Add(addPiece(piecePath, square))
			}
		}

		myWindow.SetContent(chessboard)
		myWindow.Resize(fyne.NewSize(8*squareSize, 8*squareSize))
	}
}

func isValidFEN(fen string) bool {
	parts := strings.Split(fen, " ")
	if len(parts) != 6 {
		return false
	}
	boardPart := parts[0]
	fenRows := strings.Split(boardPart, "/")
	if len(fenRows) != 8 {
		return false
	}
	for _, row := range fenRows {
		for _, char := range row {
			if !(char >= '1' && char <= '8' || strings.ContainsRune("rnbqkpRNBQKP", char)) {
				return false
			}
		}
	}
	return true
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("Chessboard")

	InitalFEN := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"
	fenChan := make(chan string)

	go update(fenChan, myWindow)
	fenChan <- InitalFEN

	go fetchData(fenChan)

	myWindow.Resize(fyne.NewSize(8*squareSize, 8*squareSize))
	myWindow.ShowAndRun()
}
