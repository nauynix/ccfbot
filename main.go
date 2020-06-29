package main

import (
	"fmt"
	"io/ioutil"
	"log"
	// "math/rand"
	"net/http"
	"net/url"
	// "os"
	"strconv"
	"strings"
	"time"
	"github.com/buger/jsonparser"
	"regexp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

func main() {
	// if os.Getenv("IS_HEROKU") != "TRUE" {
	// 	loadEnvironmentalVariables()
	// }

	// //set up telegram info
	// bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	// errCheck(err, "Failed to start telegram bot")
	// log.Printf("Authorized on account %s", bot.Self.UserName)
	// chatIDs, err := parseChatIDList((os.Getenv("CHAT_ID")))
	// errCheck(err, "Failed to fetch chat IDs")

	client := &http.Client{}
	// tgclient := AlertService{Bot: bot, ReceiverIDs: chatIDs}

	// //for heroku
	// go func() {
	// 	http.ListenAndServe(":"+os.Getenv("PORT"),
	// 		http.HandlerFunc(http.NotFound))
	// }()
	// log.Println("Logging in")
		// logIn(os.Getenv("LOGIN_ID"), os.Getenv("PASSWORD"), client)

		//fetching the booking page (client now has cookie stored inside a jar)
	log.Println("Fetching booking page")
	rawPage := slotPage(client)
    // fmt.Println(rawPage)
	// for {
	// 	//fetching session ID cookie
	// 	log.Println("Logging in")
	// 	logIn(os.Getenv("LOGIN_ID"), os.Getenv("PASSWORD"), client)

	log.Println("Parsing booking page")
	slots := extractDates(rawPage)
	_ = slots
	// valids := validSlots(slots)

	// for _, validSlot := range valids { //for all the slots which meet the rule (i.e. within 10 days of now)
	// 	fmt.Println("Slot available on " + validSlot.Date.Format("_2 Jan 2006 (Mon)") + " " + os.Getenv("SESSION_"+validSlot.SessionNumber))
	// 	// tgclient.MessageAll("Slot available on " + validSlot.Date.Format("_2 Jan 2006 (Mon)") + " " + os.Getenv("SESSION_"+validSlot.SessionNumber))
	// }
	// 	if len(valids) != 0 {
	// 		tgclient.MessageAll("Finished getting slots")
	// 	}

	// 	r := rand.Intn(300) + 120
	// 	time.Sleep(time.Duration(r) * time.Second)
	// }

}

func parseChatIDList(list string) ([]int64, error) {
	chatIDStrings := strings.Split(list, ",")
	chatIDs := make([]int64, len(chatIDStrings))
	for i, chatIDString := range chatIDStrings {
		chatID, err := strconv.ParseInt(strings.TrimSpace(chatIDString), 10, 64)
		chatIDs[i] = chatID
		if err != nil {
			return nil, err
		}
	}
	return chatIDs, nil
}

func alert(msg string, bot *tgbotapi.BotAPI, chatID int64) {
	telegramMsg := tgbotapi.NewMessage(chatID, msg)
	bot.Send(telegramMsg)
	log.Println("Sent message to " + strconv.FormatInt(chatID, 10) + ": " + msg)
}

// AlertService is a service for alerting many telegram users
type AlertService struct {
	Bot         *tgbotapi.BotAPI
	ReceiverIDs []int64
}

// Sends a message to all chats in the alert service
func (as *AlertService) MessageAll(msg string) {
	for _, chatID := range as.ReceiverIDs {
		alert(msg, as.Bot, chatID)
	}
}

func loadEnvironmentalVariables() {
	err := godotenv.Load()
	if err != nil {
		log.Print("Error loading environmental variables: ")
		log.Fatal(err.Error())
	}
}

// Returns which of the slots the user should be alerted about (ie valid slots)
func validSlots(slots []DrivingSlot) []DrivingSlot {
	valid := make([]DrivingSlot, 0)

	for _, slot := range slots {
		if slot.Date.Sub(time.Now()) < 10*(24*time.Hour) { //if slot is within 10 days of now
			valid = append(valid, slot)
		}
	}

	return valid
}

type myjar struct {
	jar map[string][]*http.Cookie
}

func (p *myjar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	fmt.Printf("The URL is : %s\n", u.String())
	fmt.Printf("The cookie being set is : %s\n", cookies)
	p.jar[u.Host] = cookies
}

func (p *myjar) Cookies(u *url.URL) []*http.Cookie {
	fmt.Printf("The URL is : %s\n", u.String())
	fmt.Printf("Cookie being returned is : %s\n", p.jar[u.Host])
	return p.jar[u.Host]
}

// // logIn logs into the CDC website, starting a session.
// // Returns the cookie storing the session data
// func logIn(learnerID string, pwd string, client *http.Client) {
// 	loginForm := url.Values{}
// 	loginForm.Add("LearnerID", learnerID)
// 	loginForm.Add("Pswd", pwd)
// 	req, err := http.NewRequest("POST", "https://www.cdc.com.sg/NewPortal/", strings.NewReader(loginForm.Encode()))
// 	errCheck(err, "Error making log in request")
// 	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/ /*;q=0.8")
// 	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
// 	req.Header.Set("Cache-Control", "no-cache")
// 	req.Header.Set("Connection", "keep-alive")
// 	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
// 	req.Header.Set("Pragma", "no-cache")
// 	req.Header.Set("Referer", "https://www.cdc.com.sg/")
// 	req.Header.Set("Upgrade-Insecure-Requests", "1")
// 	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:68.0) Gecko/20100101 Firefox/68.0")
// 	jar := &myjar{}
// 	jar.jar = make(map[string][]*http.Cookie)
// 	client.Jar = jar
// 	_, err = client.Do(req)

// 	errCheck(err, "Error logging in and getting session cookie")
// }

// Returns the page containing all the slot information
func slotPage(client *http.Client) string {
	date := "2020-07-01"
	reqBody := strings.NewReader("PreventChromeAutocomplete=&random=5efa090a8da6c&iframeid=rgpiframe5efa01656ab0d&mode=e&fctrl_1=offering_guid&offering_guid=7f3e222bae344639836173664e9772ea&fctrl_2=course_guid&course_guid=9287130353cb10164164660d58b91f7beafa9a37&fctrl_3=limited_to_course_guid_for_offering_guid_7f3e222bae344639836173664e9772ea&limited_to_course_guid_for_offering_guid_7f3e222bae344639836173664e9772ea=&fctrl_4=show_date&show_date="+date+"&ftagname_0_pcount-pid-1-361000=pcount&ftagval_0_pcount-pid-1-361000=1&ftagname_1_pcount-pid-1-361000=pid&ftagval_1_pcount-pid-1-361000=361000&fctrl_5=pcount-pid-1-361000&pcount-pid-1-361000=1&ftagname_0_pcount-pid-1-361001=pcount&ftagval_0_pcount-pid-1-361001=1&ftagname_1_pcount-pid-1-361001=pid&ftagval_1_pcount-pid-1-361001=361001&fctrl_6=pcount-pid-1-361001&pcount-pid-1-361001=0&ftagname_0_pcount-pid-1-361002=pcount&ftagval_0_pcount-pid-1-361002=1&ftagname_1_pcount-pid-1-361002=pid&ftagval_1_pcount-pid-1-361002=361002&fctrl_7=pcount-pid-1-361002&pcount-pid-1-361002=0&ftagname_0_pcount-pid-1-361003=pcount&ftagval_0_pcount-pid-1-361003=1&ftagname_1_pcount-pid-1-361003=pid&ftagval_1_pcount-pid-1-361003=361003&fctrl_8=pcount-pid-1-361003&pcount-pid-1-361003=0&ftagname_0_pcount-pid-1-361004=pcount&ftagval_0_pcount-pid-1-361004=1&ftagname_1_pcount-pid-1-361004=pid&ftagval_1_pcount-pid-1-361004=361004&fctrl_9=pcount-pid-1-361004&pcount-pid-1-361004=0:");
	req, err := http.NewRequest("POST", "https://app.rockgympro.com/b/widget/?a=equery", reqBody)
	errCheck(err, "Error querying for dates")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := client.Do(req)
	errCheck(err, "Error getting booking slots")
	bytes, err := ioutil.ReadAll(resp.Body)
	errCheck(err, "Error reading request body")
	b, err := jsonparser.GetString(bytes, "event_list_html")
	errCheck(err, "Error converting to table")
	return b
}

// DrivingSlot represents a CDC slot to go for driving lessons
type DrivingSlot struct {
	Date          time.Time
	Avability string
}

// Given the output of the slot page, finds the
func extractDates(slotPage string) []DrivingSlot {
	daySections := strings.Split(slotPage, "</tr>\n<tr>")[1:]
	slots := make([]DrivingSlot, 0)
	/* What a section looks like
      <td class='offering-page-schedule-list-time-column'>\nTue, June 30, 11 AM to  1:20 PM\n</td>
      \n
      <td>
         \n<strong>Availability</strong><br>
         <div class='offering-page-event-is-full'>Full.&nbsp;Please make a different selection.</div>
         \n
      </td>
      \n
      <td>\n\n</td>
      \n
      <td>
         \n<!--cg: 928713030419de9b236796338173310b002db345-->\n\n
      </td>
	*/
	for _, daySection := range daySections {
		// Get date
		re := regexp.MustCompile(`(\w+, \w+ \w+, \w+:\w+ \w+)`)
		dateString := string(re.FindString(strings.Split(strings.Split(daySection, "</td>")[0], ">")[1]))
		if dateString == ""{
			continue
		}
		fmt.Println(dateString)
		date, err := time.Parse("Mon, January 2, 3:04 PM", dateString)
		errCheck(err, "Error parsing date and time of slot")
		date = date.AddDate(2020, 0, 0)
		// Get avability
		re = regexp.MustCompile(`Available|[\d]* spaces?|Full`)
		rawSlots := re.FindString(daySection)
		// fmt.Println(date)
		// fmt.Println(rawSlots)
		log.Println(date, rawSlots)
		slots = append(slots, DrivingSlot{
			Date:          date,
			Avability: rawSlots,
		})
}
	return slots
}

// Returns true if a given raw slot is open
func openSlot(rawSlot string) bool {
	return strings.Contains(rawSlot, "Images1.gif")
}

func errCheck(err error, msg string) {
	if err != nil {
		log.Fatal(msg + ": " + err.Error())
	}
}
