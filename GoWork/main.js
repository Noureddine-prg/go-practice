// package main

// import (
// 	"fmt"
// 	"time"
// )

// //structs - fields initialized here
// //interface - methods initialized here

// type scraper interface {
// 	Interval() time.Duration
// 	SportsBook() string
// }

// type draftKings struct {
// 	interval   time.Duration
// 	sportsbook string
// }

// func (dk *draftKings) Interval() time.Duration {
// 	return dk.interval
// }

// func (dk *draftKings) SportsBook() string {
// 	return dk.sportsbook
// }

// type fanduel struct {
// 	interval   time.Duration
// 	sportsbook string
// }

// func (fd *fanduel) Interval() time.Duration {
// 	return fd.interval
// }

// func (fd *fanduel) SportsBook() string {
// 	return fd.sportsbook
// }

// func main() {

// 	//create new objects
// 	dkng := &draftKings{
// 		interval:   10 * time.Second,
// 		sportsbook: "draftkings",
// 	}

// 	fd := &fanduel{
// 		interval:   10 * time.Second,
// 		sportsbook: "fanduel",
// 	}

// 	scrapers := []scraper{dkng, fd}

// 	fmt.Println(scrapers[0])
// }
