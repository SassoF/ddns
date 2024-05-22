package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var (
	DB *sql.DB
)

// initialize database
func InitDatabse() (*sql.DB, error) {
	DB, err := sql.Open("mysql", "username:password@tcp(localhost:3306)/ddnsDatabase")
	if err != nil {
		return nil, err
	}
	if err = DB.Ping(); err != nil {
		return nil, err
	}

	createDDNSTable := `
		CREATE TABLE IF NOT EXISTS ddns (
			token TEXT NOT NULL UNIQUE,
			subDomain TEXT UNIQUE,
			ip TEXT,
			lastUpdate DATETIME
		);
	`
	_, err = DB.Exec(createDDNSTable)
	if err != nil {
		return nil, err
	}

	createAttemptsTable := `
		CREATE TABLE IF NOT EXISTS users (
			ip TEXT UNIQUE,
			attemptsNewDomain TEXT NOT NULL,
			lastAttemptNewDomain DATETIME NOT NULL,
			attemptsUpdate TEXT NOT NULL,
			lastAttemptUpdate DATETIME NOT NULL
		);
	`
	_, err = DB.Exec(createAttemptsTable)
	if err != nil {
		return nil, err
	}

	return DB, nil
}

// generate token base 64 with length 30
func generateToken() (string, error) {

	bytes := make([]byte, 30)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// Takes the corresponding ip, if any, to the subdomain
func GetIP(subDomain string) (string, error) {
	var ip string
	err := DB.QueryRow("SELECT ip FROM ddns WHERE subDomain = ?", subDomain).Scan(&ip)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		} else {
			return "", errors.New("Internal error")
		}
	}
	return ip, nil
}

// Adds the subdomain with maximum 3 attempts to generate the unique token
func AddDomain(subDomain, ip string) (string, error) {

	err := checkAndCreateUser(ip)
	if err != nil {
		return "", err
	}

	lessThreeAttempts, err := lessThanThreeAttempts(ip, false)
	if err != nil {
		return "", err
	}
	if !lessThreeAttempts {
		return "", errors.New("You created too many subDomains for today, try tomorrow")
	}

	const maxAttempts = 3
	var token string

	for _ = range maxAttempts {
		var err error
		token, err = generateToken()
		if err != nil {
			return "", err
		}

		// Attempt to insert the token into the database
		_, err = DB.Exec("INSERT INTO ddns (token, subDomain, ip, lastUpdate) VALUES (?, ?, ?, ?)",
			token, subDomain, ip, time.Now())
		if err == nil {
			// Successfully inserted token
			err = updateAttempts(ip, false)
			if err != nil {
				log.Println(err)
			}
			return token, nil
		}

		// Check for MySQL specific errors
		if mysqlErr, ok := err.(*mysql.MySQLError); ok {
			if mysqlErr.Number == 1062 {
				if strings.Contains(mysqlErr.Message, "token") {
					// Token already exists, try again
					log.Println("Token already exists")
					continue
				} else if strings.Contains(mysqlErr.Message, "subDomain") {
					return "", errors.New("subDomain alredy exixst")
				}
			}
		}

		// For other errors, log and return
		log.Println("Database error:", err)
		return "", err
	}

	// Max attempts reached, return error
	return "", errors.New("max attempts reached")
}

// updates the ip of the passed subdomain if the passed token is correct
func UpdateDomain(token, subDomain, userIP, newIP string) error {

	lessThreeAttempts, err := lessThanThreeAttempts(userIP, true)
	if err != nil {
		return err
	}
	if !lessThreeAttempts {
		return errors.New("Too many attempts, retry in 5 minutes")
	}

	var DBToken, DBIP string

	err = DB.QueryRow("SELECT token, ip FROM ddns WHERE subDomain = ?", subDomain).Scan(&DBToken, &DBIP)

	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("subDomain doesn't exixst")
		} else {
			log.Println(err.Error())
			return errors.New("Internal error")
		}
	}
	if token != DBToken {
		err := updateAttempts(userIP, true)
		if err != nil {
			log.Println("updateAttempts", err.Error())
		}
		return errors.New("The token isn't valid")
	}

	stmt, err := DB.Prepare("UPDATE ddns SET ip = ? WHERE subDomain = ?")
	if err != nil {
		log.Println(err.Error())
		return errors.New("Internal error")
	}
	defer stmt.Close()

	result, err := stmt.Exec(newIP, subDomain)
	if err != nil {
		log.Println(err.Error())
		return errors.New("Internal error")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err.Error())
		return errors.New("Internal error")
	}
	if rowsAffected == 0 {
		//No update, same ip
	}

	return nil
}
