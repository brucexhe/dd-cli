package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	http.HandleFunc("/image", imageHandler)
	http.HandleFunc("/deploy-file", deployFileHandler)
	http.HandleFunc("/hash", hashHandler)
	http.HandleFunc("/deploy", deployHandler)

	log.Println("Starting dd-server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "Missing service parameter", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpFile, err := os.CreateTemp("", "image-*.tar")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("docker", "load", "-i", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, string(output), http.StatusInternalServerError)
		return
	}

	log.Printf("Image loaded: %s", output)
	w.Write([]byte("Image uploaded successfully"))
}

func deployFileHandler(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "Missing service parameter", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	dir := filepath.Join("deployments", service)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	dstPath := filepath.Join(dir, "deploy.yml")
	dstFile, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Deployment file saved: %s", dstPath)
	w.Write([]byte("Deployment file uploaded"))
}

func hashHandler(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "Missing service parameter", http.StatusBadRequest)
		return
	}

	path := filepath.Join("deployments", service, "deploy.yml")
	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte(hex.EncodeToString(hasher.Sum(nil)))
}

func deployHandler(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		http.Error(w, "Missing service parameter", http.StatusBadRequest)
		return
	}

	path := filepath.Join("deployments", service, "deploy.yml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "Deployment file not found", http.StatusNotFound)
		return
	}

	cmd := exec.Command("docker", "stack", "deploy", "-c", path, service)
	output, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, string(output), http.StatusInternalServerError)
		return
	}

	log.Printf("Deployment output: %s", output)
	w.Write([]byte("Deployment successful"))
}