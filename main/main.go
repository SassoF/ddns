package main

import (
	"ddnsProject/database"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
)

func main() {
	var err error
	database.DB, err = database.InitDatabse()
	if err != nil {
		log.Println(err)
		return
	}
	defer database.DB.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/newDomain", newDomainHandler)
	mux.HandleFunc("/update", updateDomainHandler)
	mux.HandleFunc("/", rootHandler)

	fmt.Println("Server listening on port 80")
	log.Fatalln(http.ListenAndServe(":80", mux))
}

// Checks if it contains a subdomain and if so redirects
func rootHandler(w http.ResponseWriter, r *http.Request) {

	parts := strings.Split(r.Host, ".")
	if len(parts) == 2 {
		fmt.Fprintf(w, "Welcome to the main domain!")
		return
	} else if len(parts) != 3 {
		fmt.Fprintln(w, "Url not valid")
		return
	}
	ip, err := database.GetIP(parts[0])
	if err != nil {
		log.Println(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ip == "" {
		fmt.Fprintln(w, "subDomain doesn't exixst")
		return
	}

	http.Redirect(w, r, "http://"+ip, http.StatusMovedPermanently)

}

// Add subDomain
func newDomainHandler(w http.ResponseWriter, r *http.Request) {
	ip := strings.Split(r.RemoteAddr, ":")[0]

	//checks if the request contains domains
	subDomain := r.URL.Query().Get("domain")
	if subDomain == "" {
		fmt.Fprintln(w, "Missing domain")
		return
	}
	//only alphanumeric characters
	regex := regexp.MustCompile("^[a-zA-Z0-9]*$")
	if !regex.MatchString(subDomain) {
		fmt.Fprintln(w, "Enter only alphanumeric characters")
		return
	}
	//max length for the domain
	if len(subDomain) >= 64 {
		fmt.Fprintln(w, "The subDomain is too long, max 64 characters")
		return
	}
	//Adds the suBdomain to the database with client ip
	token, err := database.AddDomain(subDomain, ip)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Domain: %s\nToken: %s\nIP: %s\n", subDomain, token, ip)

}

// Update subDomain with token domain ip, ip option
func updateDomainHandler(w http.ResponseWriter, r *http.Request) {

	userIP := strings.Split(r.RemoteAddr, ":")[0]

	subDomain := r.URL.Query().Get("domain")
	token := r.URL.Query().Get("token")
	subDomainIP := r.URL.Query().Get("ip")

	if subDomainIP == "" {
		subDomainIP = userIP
	} else if subDomainIP := net.ParseIP(subDomainIP); subDomainIP == nil {
		fmt.Println(subDomainIP)
		fmt.Fprintln(w, "Ip not valid")
		return
	}
	if subDomain == "" {
		fmt.Fprintln(w, "Domain missing")
		return
	}
	if token == "" {
		fmt.Fprintln(w, "Token missing")
		return
	}

	err := database.UpdateDomain(token, subDomain, userIP, subDomainIP)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	fmt.Fprintln(w, "Update successful")
	//fmt.Printf("%s %s %s\n", subDomain, token, userIP, subDomainIP)
}
