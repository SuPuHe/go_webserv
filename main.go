package main

import (
	"fmt"
	"log"
	"net/http"
)

func apiHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		fmt.Fprintln(w, "GET method. You can get data here")
	case http.MethodPost:
		fmt.Fprintln(w, "POST methond. You can save file here")
	case http.MethodDelete:
		fmt.Fprintln(w, "Delete Method. File is deleted")
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {

	cfg, err := LoadConfig("config/default.toml")
	if err != nil {
		log.Fatalf("Error with loading config file: %v", err)
	}

	cfg.PrettyPrint()

	mux := http.NewServeMux();

	mux.Handle("/", http.FileServer(http.Dir("./www")))

	// mux.HandleFunc("/api", apiHandler)

	fmt.Println("Server is running: http://localhost:8080")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Sever running error: %v\n", err)
	}
}
