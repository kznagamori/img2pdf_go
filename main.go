package main

import (
	"bytes"
	"image"
	"image/jpeg"
	_ "image/png"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/jung-kurt/gofpdf"
	"golang.org/x/image/webp"
)

func main() {
	// カレントディレクトリの取得
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("カレントディレクトリの取得に失敗しました: %v", err)
	}
	dirName := filepath.Base(dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("ディレクトリの読み込みに失敗しました: %v", err)
	}
	files := make([]fs.FileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			log.Fatalf("ファイル情報の取得に失敗しました: %v", err)
		}
		files = append(files, info)
	}
	// ファイル名でソート
	sort.Slice(files, func(i, j int) bool {
		return less(files[i].Name(), files[j].Name())
	})

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size:    gofpdf.SizeType{Wd: 0, Ht: 0}, // サイズは後で設定する
	})
	pdf.SetAutoPageBreak(false, 0)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") || strings.HasSuffix(filename, ".png") || strings.HasSuffix(filename, ".webp") {
			if err := addImageToPDF(pdf, filename); err != nil {
				log.Printf("画像の追加に失敗しました: %v", err)
				continue
			}
		}
	}

	outputFilePath := filepath.Join(dir, dirName+".pdf")
	err = pdf.OutputFileAndClose(outputFilePath)
	if err != nil {
		log.Fatalf("PDFファイルの作成に失敗しました: %v", err)
	}
	log.Printf("PDFファイルが生成されました: %s", outputFilePath)
}

func addImageToPDF(pdf *gofpdf.Fpdf, filename string) error {
	imgFile, err := os.Open(filename)
	log.Printf("画像:%v追加します", filename)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	var img image.Image
	// JPEGデータをメモリ内で生成
	var buf bytes.Buffer
	if strings.HasSuffix(filename, ".webp") {
		var err error
		img, err = webp.Decode(imgFile)
		if err != nil {
			return err
		}
	} else {
		var err error
		img, _, err = image.Decode(imgFile)
		if err != nil {
			return err
		}
	}
	opt := jpeg.Options{Quality: 75} // JPEGの圧縮率
	if err := jpeg.Encode(&buf, img, &opt); err != nil {
		return err
	}

	// 画像サイズに基づいてページを追加
	width := float64(img.Bounds().Dx()) * 0.264583 // mmに変換 (1 inch = 25.4 mm)
	height := float64(img.Bounds().Dy()) * 0.264583
	pdf.AddPageFormat("P", gofpdf.SizeType{Wd: width, Ht: height})

	// メモリバッファから画像をPDFに追加
	pdf.RegisterImageOptionsReader(filename, gofpdf.ImageOptions{ImageType: "JPEG", ReadDpi: true}, &buf)
	pdf.ImageOptions(filename, 0, 0, width, height, false, gofpdf.ImageOptions{}, 0, "")
	return nil
}

// less はファイル名を解析して比較する
func less(name1, name2 string) bool {
	// 数値部分と文字列部分に分割
	parts1 := splitName(name1)
	parts2 := splitName(name2)

	// 最小の長さを取得
	minLen := len(parts1)
	if len(parts2) < minLen {
		minLen = len(parts2)
	}

	for i := 0; i < minLen; i++ {
		if parts1[i] != parts2[i] {
			// 数値部分は数値として比較
			if num1, err1 := strconv.Atoi(parts1[i]); err1 == nil {
				if num2, err2 := strconv.Atoi(parts2[i]); err2 == nil {
					return num1 < num2
				}
			}
			// 文字列部分は文字列として比較
			return parts1[i] < parts2[i]
		}
	}

	// 全て同じ場合は、短い方が先
	return len(parts1) < len(parts2)
}

// splitName はファイル名を数値と非数値部分に分割する
func splitName(name string) []string {
	var parts []string
	var currentPart strings.Builder

	for _, runeValue := range name {
		if runeValue >= '0' && runeValue <= '9' {
			if currentPart.Len() > 0 && !unicode.IsDigit(rune(currentPart.String()[0])) {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			}
		} else {
			if currentPart.Len() > 0 && unicode.IsDigit(rune(currentPart.String()[0])) {
				parts = append(parts, currentPart.String())
				currentPart.Reset()
			}
		}
		currentPart.WriteRune(runeValue)
	}

	if currentPart.Len() > 0 {
		parts = append(parts, currentPart.String())
	}

	return parts
}
