package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 4 || os.Args[1] != "-f" {
		log.Fatal("Usage: dd -f <deploy.yml> <service-name>")
	}

	deployFile := os.Args[2]
	serviceName := os.Args[3]

	// Parse deploy.yml to get image name
	imageName, err := parseDeployYAML(deployFile)
	if err != nil {
		log.Fatalf("Error parsing deploy.yml: %v", err)
	}

	// Docker build
	log.Printf("Building Docker image: %s", imageName)
	if err := dockerBuild(imageName); err != nil {
		log.Fatalf("Docker build failed: %v", err)
	}

	// Save Docker image
	tmpFile, err := os.CreateTemp("", "dd-image-*.tar")
	if err != nil {
		log.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	log.Printf("Saving image to %s", tmpFile.Name())
	if err := dockerSave(imageName, tmpFile.Name()); err != nil {
		log.Fatal(err)
	}

	// Upload image
	log.Printf("Uploading image to server")
	if err := uploadImage(tmpFile.Name(), serviceName); err != nil {
		log.Fatal(err)
	}

	// Check and upload deploy.yml
	localHash, err := fileHash(deployFile)
	if err != nil {
		log.Fatal(err)
	}

	remoteHash, err := getRemoteHash(serviceName)
	if err != nil {
		log.Printf("Could not get remote hash: %v", err)
	}

	if localHash != remoteHash {
		log.Printf("Uploading updated deploy.yml")
		if err := uploadDeployFile(deployFile, serviceName); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Printf("No changes in deploy.yml")
	}

	// Trigger deployment
	log.Printf("Triggering deployment")
	if err := triggerDeploy(serviceName); err != nil {
		log.Fatal(err)
	}

	log.Println("Deployment complete!")
}

func parseDeployYAML(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var config struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", err
	}

	for _, service := range config.Services {
		return service.Image, nil
	}

	return "", fmt.Errorf("no image found in deploy.yml")
}

func dockerBuild(image string) error {
	cmd := exec.Command("docker", "build", "-t", image, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dockerSave(image, path string) error {
	cmd := exec.Command("docker", "save", "-o", path, image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func uploadImage(path, service string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", filepath.Base(path))
	io.Copy(part, file)
	writer.Close()

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("http://localhost:8080/image?service=%s", service),
		body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed: %s", resp.Status)
	}
	return nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func getRemoteHash(service string) (string, error) {
	resp, err := http.Get(
		fmt.Sprintf("http://localhost:8080/hash?service=%s", service),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status: %s", resp.Status)
	}

	hash, _ := io.ReadAll(resp.Body)
	return string(hash), nil
}

func uploadDeployFile(path, service string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "deploy.yml")
	io.Copy(part, file)
	writer.Close()

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("http://localhost:8080/deploy-file?service=%s", service),
		body,
	)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed: %s", resp.Status)
	}
	return nil
}

func triggerDeploy(service string) error {
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:8080/deploy?service=%s", service),
		"text/plain",
		nil,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("deploy failed: %s", resp.Status)
	}
	return nil
}