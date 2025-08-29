package PythonScrapper

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func HandleScrapeKeyInvoiceProducts(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Scrape Key Invoice Products request")

	cwd, _ := os.Getwd()
	scriptPath := filepath.Join(cwd, "pkg", "api", "server", "PythonScrapper", "python", "sync-products.py")

	cmd := exec.Command("py", scriptPath) // use "py" on Windows locally
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Println("Error: ", err)
	}

	fmt.Println(string(output))
}
