package main

import (
	"flag"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/chai2010/webp"
)

var (
	qualityFlag   = flag.Int("quality", 90, "Quality of the converted images")
	cacheFolder   = flag.String("cache", "./cache", "Folder to cache images")
	cacheClearKey = flag.String("cachekey", "defaultKey", "Key to clear cache")
)

func imageHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Path
	cachePath := *cacheFolder + filePath + ".webp"

	if _, err := os.Stat(cachePath); err == nil {
		http.ServeFile(w, r, cachePath)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	contentType := http.DetectContentType(buffer)

	_, err = file.Seek(0, 0)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	var img image.Image
	switch {
	case strings.Contains(contentType, "jpeg"):
		img, err = jpeg.Decode(file)
	case strings.Contains(contentType, "png"):
		img, err = png.Decode(file)
	case strings.HasPrefix(contentType, "image/webp"):
		http.ServeFile(w, r, filePath)
		return
	default:
		http.Error(w, "Unsupported file type", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Error processing image", http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		http.Error(w, "Error creating cache directory", http.StatusInternalServerError)
		return
	}

	cacheFile, err := os.Create(cachePath)
	if err != nil {
		http.Error(w, "Error creating cache file", http.StatusInternalServerError)
		return
	}
	defer cacheFile.Close()

	w.Header().Set("Content-Type", "image/webp")
	err = webp.Encode(cacheFile, img, &webp.Options{Quality: float32(*qualityFlag)})
	if err != nil {
		http.Error(w, "Error encoding image", http.StatusInternalServerError)
		return
	}

	http.ServeFile(w, r, cachePath)
}

func cacheClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("key") != *cacheClearKey {
		http.Error(w, "Invalid cache clear key", http.StatusForbidden)
		return
	}

	if err := os.RemoveAll(*cacheFolder); err != nil {
		http.Error(w, "Error clearing cache", http.StatusInternalServerError)
		return
	}

	if _, err := w.Write([]byte("Cache cleared successfully")); err != nil {
		http.Error(w, "Error writing response", http.StatusInternalServerError)
		return
	}
}

func main() {
	flag.Parse()

	if err := os.MkdirAll(*cacheFolder, 0755); err != nil {
		log.Fatal("Error creating cache folder:", err)
	}

	http.HandleFunc("/", imageHandler)
	http.HandleFunc("/clearcache", cacheClearHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
