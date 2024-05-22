// loginDatabase
package database

import (
	"database/sql"
	"errors"
	"log"
	"time"
)

const (
	QUERY_SELECT_ATTEMPTS_UPDATE       = "SELECT attemptsUpdate FROM users WHERE ip = ?"
	QUERY_SELECT_ATTEMPTS_NEWDOMAIN    = "SELECT attemptsNewDomain FROM users WHERE ip = ?"
	QUERY_UPDATE_ATTEMPTS_UPDATE       = "UPDATE users SET attemptsUpdate = ?, lastAttemptUpdate = ? WHERE ip = ?"
	QUERY_UPDATE_ATTEMPTS_NEWDOMAIN    = "UPDATE users SET attemptsNewDomain = ?, lastAttemptNewDomain = ? WHERE ip = ?"
	QUERY_RESET_ATTEMPTS_UPDATE        = "UPDATE users SET attemptsUpdate = 0 WHERE ip = ?"
	QUERY_RESET_ATTEMPT_NEWDOMAIN      = "UPDATE users SET attemptsNewDomain = 0 WHERE ip = ?"
	QUERY_SELECT_LASTATTEMPT_UPDATE    = "SELECT lastAttemptUpdate FROM users WHERE ip = ?"
	QUERY_SELECT_LASTATTEMPT_NEWDOMAIN = "SELECT lastAttemptNewDomain FROM users WHERE ip = ?"
)

// update the attempts by one
func updateAttempts(ip string, update bool) error {
	var query1, query2 string
	if update {
		query1 = QUERY_SELECT_ATTEMPTS_UPDATE
		query2 = QUERY_UPDATE_ATTEMPTS_UPDATE
	} else {
		query1 = QUERY_SELECT_ATTEMPTS_NEWDOMAIN
		query2 = QUERY_UPDATE_ATTEMPTS_NEWDOMAIN
	}

	err := checkUser(ip)
	if err != nil {
		return err
	}

	// Query to get the current value of attempt
	var currentAttempt int
	err = DB.QueryRow(query1, ip).Scan(&currentAttempt)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("IP not found")
		} else {
			log.Println(err.Error())
			return errors.New("Internal error")
		}
	}

	// Execution of update instruction
	stmt, err := DB.Prepare(query2)
	if err != nil {
		log.Println(err.Error())
		return errors.New("Errore interno")
	}
	defer stmt.Close()

	result, err := stmt.Exec(currentAttempt+1, time.Now(), ip)
	if err != nil {
		log.Println(err.Error())
		return errors.New("Errore interno")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err.Error())
		return errors.New("Errore interno")
	}
	if rowsAffected == 0 {
		log.Println("Nessuna riga aggiornata")
		return errors.New("Aggiornamento non riuscito")
	}

	return nil
}

// Create user if not exists
func checkUser(ip string) error {
	// Check if the users exists
	var exists bool
	err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE ip = ?)", ip).Scan(&exists)
	if err != nil {
		return err
	}

	// If not exists create a user
	if !exists {
		_, err := DB.Exec(
			"INSERT INTO users "+
				"(ip, attemptsNewDomain, lastAttemptNewDomain, "+
				"attemptsUpdate, lastAttemptUpdate) VALUES (?, 0, ?, 0, ?)",
			ip, time.Now(), time.Now())

		if err != nil {
			return err
		}
	}
	return nil

}

// Check if the user has less than three attemtps
// Check if 5 minutes have passed and if so, reset the attempts
func lessThanThreeAttempts(ip string, update bool) (bool, error) {
	var query1, query2 string
	var timeToWait time.Duration
	if update {
		query1 = QUERY_SELECT_ATTEMPTS_UPDATE
		query2 = QUERY_SELECT_LASTATTEMPT_UPDATE
		timeToWait = 5
	} else {
		query1 = QUERY_SELECT_ATTEMPTS_NEWDOMAIN
		query2 = QUERY_SELECT_LASTATTEMPT_NEWDOMAIN
		timeToWait = 1440
	}

	err := checkUser(ip)
	if err != nil {
		return false, err
	}

	var attempts int
	err = DB.QueryRow(query1, ip).Scan(&attempts)
	if err != nil {
		log.Println(err.Error())
		return false, errors.New("Errore interno")
	}
	if attempts >= 3 {
		var lastAttemptStr string
		err = DB.QueryRow(query2, ip).Scan(&lastAttemptStr)
		if err != nil {
			log.Println(err.Error())
			return false, errors.New("Errore interno")
		}

		lastAttempt, err := time.Parse("2006-01-02 15:04:05", lastAttemptStr)
		if err != nil {
			log.Println(err.Error())
			return false, errors.New("Error parsing time")
		}

		//check whether 5 minutes have passed if there were 3 attempts

		if time.Now().Sub(lastAttempt) < timeToWait*time.Minute {
			return false, nil
		} else {
			if err := resetAttemptsUpdate(ip, update); err != nil {
				log.Println("Reset attemtps", err.Error())
				return false, errors.New("Internal error")
			}
		}
	}

	return true, nil
}

// if 5 minutes have passed, the attempts are reset to 0
func resetAttemptsUpdate(ip string, update bool) error {
	var query string
	if update {
		query = QUERY_RESET_ATTEMPTS_UPDATE
	} else {
		query = QUERY_RESET_ATTEMPT_NEWDOMAIN
	}

	stmt, err := DB.Prepare(query)
	if err != nil {
		log.Println(err.Error())
		return errors.New("Internal error")
	}
	defer stmt.Close()

	result, err := stmt.Exec(ip)
	if err != nil {
		log.Println(err.Error())
		return errors.New("Internal error")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Println(err.Error())
		return errors.New("Internal Error")
	}
	if rowsAffected == 0 {
		log.Println("No reset attempts", ip)
		return errors.New("Reset attemtps failed")
	}

	return nil
}
