package main

import (
	"encoding/hex"
	"fmt"
	"github.com/Sam-Arch/hashem"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"net/http"
	"strconv"
)

type User struct {
	Id     string `bson:"id"`
	Day    int    `bson:"day"`
	Name   string `bson: "name"`
	Visit  int    `bson: "visit"`
}

type Nicks struct {
	takenNick bool
}

///
//session handlers and session clear'ers lol
var cookieHandler = securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32))
var router = mux.NewRouter()
var sessName string
var count int = 0
var myDay int
var myName string
var currUser string
var visitCount int

func getRandomSessName(str string) {
	sessName = hex.EncodeToString([]byte(str))
}

func getUserName(w http.ResponseWriter, r *http.Request) (userName string) {
	if cookie, err := r.Cookie("session"); err == nil {
		cookieValue := make(map[string]string)
		if err = cookieHandler.Decode("session", cookie.Value, &cookieValue); err == nil {
			userName = cookieValue["name"]
		}
	}
	return userName
}

func setSession(userName string, w http.ResponseWriter) {
	getRandomSessName(userName)
	value := map[string]string{"name": userName}
	if encoded, err := cookieHandler.Encode("session", value); err == nil {
		cookie := &http.Cookie{Name: "session", Value: encoded, Path: "/"}
		http.SetCookie(w, cookie)
	}
}

func clearSession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)
}

////end of session and session clear handlers

//serve index page
func indexPagehandler(w http.ResponseWriter, r *http.Request) {
	template.Must(template.ParseFiles("index.html")).Execute(w, nil)
}

//login and logout handlers
func loginHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	myNameHere := getUserName(w,r)
	fmt.Println("he is called", myNameHere)
	password := r.FormValue("password")
	hashPassword, hashSalt := giveBytes(password, name)
	hash := hashem.HashPassword(hashPassword, hashSalt)

	isValidHash := checkUser(hash)
	redirectTarget := "/"

	if name != "" && password != "" {
		if isValidHash {

			setSession(name, w)
			UserData := getDay(hash)
			myDay = UserData.Day
			myName = UserData.Name
			myVisit := UserData.Visit
			if myDay == 21 && myVisit == 2{
			clearSession(w)
			}
			UserData.dayHandler(w)
		} else {
			http.Redirect(w, r, "/reg", 302)
		}
	} else {
		http.Redirect(w, r, redirectTarget, 302)
	}
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("my name is ", myName)
	if true := updateVisit(myName); true {
		fmt.Println("Visit updated")
	} else {
		fmt.Println("oopsy")
	}
	
	//update count here somehow
	http.Redirect(w, r, "/", 302)
}

//end of login and logout handlers

//database code to check if user exists
//returns true if user exists
func checkUser(hash string) bool {
	session, err := mgo.Dial("localhost")
	if err != nil {

		panic(err)
	}
	defer session.Close()
	c := session.DB("test").C("users")
	result := User{}

	err = c.Find(bson.M{"id": hash}).One(&result)
	if err != nil {

		return false
	}
	return true
}

//add new user
func (u *User) addUser() bool {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c := session.DB("test").C("users")
	result := User{}
	err = c.Find(bson.M{"id": u.Id}).One(&result)
	//		&User{"completed", "2", "JanyGwan"})

	if err != nil {
		err = c.Insert(&User{u.Id, u.Day, u.Name, u.Visit})
		if err != nil {
			return false //user added successfully

		}
	}
	fmt.Println("i just added", u.Name)
	return true
}

//check if nick is taken
//add new user
func checkNick(name string) bool {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c := session.DB("test").C("users")
	result := Nicks{}
	err = c.Find(bson.M{"name": name}).One(&result)
	if err != nil {

		return false //name doesnt exist
	}

	return true //name exists,don't add user
}

//end of nick checker

//update handler. updates the day
func updateDay(name string) bool {
	session, err := mgo.Dial("localhost")
	if err != nil {

		panic(err)
	}
	defer session.Close()
	c := session.DB("test").C("users")
	result := User{}

	err = c.Find(bson.M{"name": name}).One(&result)
	if err != nil {

		return false
	}
	day := result.Day

	if day < 21 {
		day = day + 1
	}
	if day > 21 { //remove this uder from the db. He has learnt what was needed. email him book
		err = c.Remove(bson.M{"name": name})
		fmt.Println("goodbye", name)
		
		return false
	}

	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"day": day}},
		ReturnNew: true,
	}
	_, err = c.Find(bson.M{"name": name}).Apply(change, &result)
	if err != nil {
		fmt.Println("oopsies")
		return false
	}

	newDayVisit := mgo.Change{
		Update:    bson.M{"$set": bson.M{"visit": 1}},
		ReturnNew: true,
	}

	_, err = c.Find(bson.M{"name": name}).Apply(newDayVisit, &result)
	if err != nil {
		fmt.Println("oopsies...couldnt update visit")
		return false
	}
	return true
}

//update visit. updates the visit
func updateVisit(name string) bool {
	session, err := mgo.Dial("localhost")
	if err != nil {

		panic(err)
	}
	defer session.Close()
	c := session.DB("test").C("users")
	result := User{}

	err = c.Find(bson.M{"name": name}).One(&result)
	if err != nil {

		return false //something went wrong
	}
	visit := result.Visit
	day := result.Day
	fmt.Println("visit == ", visit)
	fmt.Println("day == ", day)
	if day < 18 {
		switch visit {
		case 1:
			visit = 2
		case 2:
			visit = 3
		case 3:
			visit = 1
			updateMyDay(name)
		default:
			return false
		}
	} else {
		fmt.Println("we are here")
		switch visit {
		case 1:
			visit = 2
    case 2:
      visit = 1
			updateMyDay(name)
		default:
			return false
		}
	}
	change := mgo.Change{
		Update:    bson.M{"$set": bson.M{"visit": visit}},
		ReturnNew: true,
	}
	_, err = c.Find(bson.M{"name": name}).Apply(change, &result)
	if err != nil {
		fmt.Println("oopsies")
		return false
	}
	fmt.Println("visit is now ", visit)
	return true
}

//gets the day for the user
func getDay(hash string) User {
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()
	c := session.DB("test").C("users")
	result := User{}
	err = c.Find(bson.M{"id": hash}).One(&result)
	if err != nil {
		fmt.Println("oh snap!!!")
	}
	return result
}

////end of getting the day



func (u *User) dayHandler(w http.ResponseWriter) {
	day := strconv.Itoa(u.Day)
	currUser = u.Name //this keeps track of the current user whose day is being rendered
	template.Must(template.ParseFiles(day+".html")).Execute(w, u)
}

func giveBytes(pass, salt string) ([]byte, []byte) {
	return []byte(pass), []byte(salt)
}

func regHandler(w http.ResponseWriter, r *http.Request) {
	renderPage(w, "reg")

}

func thankYouHandler(w http.ResponseWriter, r *http.Request) {
	password := r.FormValue("password")
	nick := r.FormValue("nick")

	if isTakenNick := checkNick(nick); isTakenNick {
		// template.Must(template.ParseFiles("reg.html")).Execute(w, m)
		takenNick := &Nicks{}
		takenNick.takenNick = true
		http.Redirect(w, r, "/reg", http.StatusFound)
	}
	count = count + 1
	hashPassword, hashSalt := giveBytes(password, nick)

	hash := hashem.HashPassword(hashPassword, hashSalt)

	if nick == "" && password == "" {
		http.Redirect(w, r, "/", http.StatusFound)
	}
	u := &User{Id: hash, Day: 1, Name: nick, Visit: 1}
	if count == 1 {
		if test := u.addUser(); test {
			myName = nick
			setSession(nick, w)
		}
	}
	u.dayHandler(w)
}

//update day

func updateMyDay(name string) {
	if done := updateDay(name); done {
		fmt.Println("done")

	} else {
		fmt.Println("Houston!!!!", name)
	}

}

//end update day
func updateHandler(w http.ResponseWriter, r *http.Request) {

	//	fmt.Println("i am ", name)
	if done := updateDay(myName); done {
		fmt.Println("done")

	} else {
		fmt.Println("Houston!!!!", myName)
	}
	http.Redirect(w, r, "/", http.StatusFound)

}

//for pages that execute with the struct
func renderTemplate(w http.ResponseWriter, page string, u *User) {
	template.Must(template.ParseFiles(page+".html")).Execute(w, u)
}

//for Pages that dont execute with the struct
func renderPage(w http.ResponseWriter, page string) {
	template.Must(template.ParseFiles(page+".html")).Execute(w, nil)
}
func wikiHandler(w http.ResponseWriter, r *http.Request) {
renderPage(w, "wiki")
}

func main() {
	router.HandleFunc("/", indexPagehandler)
	router.HandleFunc("/wiki", wikiHandler)
	router.HandleFunc("/login", loginHandler).Methods("POST")
	router.HandleFunc("/logout", logoutHandler).Methods("POST")
	router.HandleFunc("/reg", regHandler)
	router.HandleFunc("/update", updateHandler).Methods("POST")
	router.HandleFunc("/thankyou", thankYouHandler).Methods("POST")
	http.Handle("/", router)
	http.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
	http.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("images"))))
	http.Handle("/fonts/", http.StripPrefix("/fonts/", http.FileServer(http.Dir("fonts"))))
	http.Handle("/sessions/", http.StripPrefix("/sessions/", http.FileServer(http.Dir("sessions"))))
	http.ListenAndServe(":8080", nil)
}
