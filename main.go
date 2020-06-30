package main

import (
	"io/ioutil"
	"log"
	"fmt"
	"math/rand"
	"net/http"
	// "net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/buger/jsonparser"
	"regexp"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

var valids []ClimbingSlot
var toBook []BookingSlot
var timeSlotWeekDay = []time.Duration{
	time.Duration(660),
	time.Duration(690),
	time.Duration(720),
	time.Duration(750),
	time.Duration(810),
	time.Duration(840),
	time.Duration(870),
	time.Duration(900),
	time.Duration(960),
	time.Duration(990),
	time.Duration(1020),
	time.Duration(1050),
	time.Duration(1110),
	time.Duration(1140),
	time.Duration(1170),
	time.Duration(1200),
	time.Duration(1230),
	time.Duration(1250),}
var timeSlotWeekEnd = []time.Duration{
	time.Duration(540),
	time.Duration(570),
	time.Duration(600),
	time.Duration(630),
	time.Duration(690),
	time.Duration(720),
	time.Duration(750),
	time.Duration(780),
	time.Duration(840),
	time.Duration(870),
	time.Duration(900),
	time.Duration(930),
	time.Duration(990),
	time.Duration(1020),
	time.Duration(1050),
	time.Duration(1080),
	time.Duration(1110),
	time.Duration(1130),}
// BtnPrev represents prev
	const BtnPrev = "<"
// BtnNext represents next
const BtnNext = ">"
const AWAITING = string("Awaiting")
const SUCCESS = string("Successful")
const FAILED = string("Failed")
const charset = "0123456789abcde"

// ClimbingSlot represents an available climbing slot
type ClimbingSlot struct {
	Date          	time.Time
	Availability 	string
	GUID 			string
}
// BookingSlot represents a slot that a user wants to book
type BookingSlot struct {
	Date			time.Time
	ChatID			int64
	Outcome			string
}

func removeSlotKeyboard(ChatID int64) tgbotapi.InlineKeyboardMarkup{
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	for _, slot := range toBook {
		text := "Remove " + slot.Date.Format("02/01 (Mon) 3:04 PM")
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(text, text)))
	}
	return keyboard
}

func createTimePicker(date time.Time, weekday bool) tgbotapi.InlineKeyboardMarkup{
	keyboard := tgbotapi.NewInlineKeyboardMarkup()
	if weekday {
		for _, slot := range timeSlotWeekDay {
			txt := date.Add(slot*time.Minute).Format("02/01 (Mon) 3:04 PM")
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(txt, txt)))
		}
	} else {
		for _, slot := range timeSlotWeekEnd {
			txt := date.Add(slot*time.Minute).Format("02/01 (Mon) 3:04 PM")
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(txt, txt)))
		}
	}
	return keyboard
}

func main() {
	//set up telegram info
	loadEnvironmentalVariables()
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	errCheck(err, "Failed to start telegram bot")
	log.Printf("Authorized on account %s", bot.Self.UserName)
	// chatIDs, err := parseChatIDList((os.Getenv("CHAT_ID")))
	errCheck(err, "Failed to fetch chat IDs")
	client := &http.Client{}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)

	// Run the telegram bot
	go runTelegramBot(updates, bot, client)
	// Run updating and making bookings
	go runUpdateBookings(client, bot)
	select{}
}

func runTelegramBot(updates tgbotapi.UpdatesChannel, bot *tgbotapi.BotAPI, client *http.Client){
	//Start telegram
	var year int
	var month time.Month
	var calendar tgbotapi.InlineKeyboardMarkup
	var dateMatch = regexp.MustCompile(`^[0-9][0-9][0-9][0-9]\.[0-9][0-9]\.[0-9][0-9]$`)
	var dateTimeMatch = regexp.MustCompile(`^\d{2}\/\d{2} .{5} .* .*$`)
	var removeDateTimeMatch = regexp.MustCompile(`^Remove \d{2}\/\d{2} .{5} .* .*$`)
	for update := range updates {
		if update.CallbackQuery != nil{
			log.Println(update)
			switch {
				case update.CallbackQuery.Data == ">":
					log.Println("Next")
					calendar, year, month = HandlerNextButton(year, month)
					msg := tgbotapi.NewEditMessageReplyMarkup(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, calendar)
					bot.Send(msg)
				case update.CallbackQuery.Data == "<":
					log.Println("Prev")
					calendar, year, month = HandlerPrevButton(year, month)
					msg := tgbotapi.NewEditMessageReplyMarkup(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, calendar)
					bot.Send(msg)
				case dateMatch.MatchString(update.CallbackQuery.Data):
					log.Println(update.CallbackQuery.Data)
					date,_ := time.Parse("2006.01.02", update.CallbackQuery.Data)
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
					if int(date.Weekday()) == 0 || int(date.Weekday()) == 6{
						// Weekend
						msg.ReplyMarkup = createTimePicker(date, false)
					} else{
						// Weekday
						msg.ReplyMarkup = createTimePicker(date, true)
					}
					bot.Send(msg)
				case removeDateTimeMatch.MatchString(update.CallbackQuery.Data):
					re := regexp.MustCompile(`\d{2}\/\d{2} .{5} .* .*$`)
					toMatchDate,_ := time.Parse("02/01 (Mon) 3:04 PM", re.FindString(update.CallbackQuery.Data))
					toMatchDate = toMatchDate.AddDate(time.Now().Year(), 0, 0)
					for i, book := range toBook{
						if book.ChatID == update.CallbackQuery.Message.Chat.ID && book.Date == toMatchDate {
							toBook = append(toBook[:i], toBook[i+1:]...)
							break
						}
					}
					bot.Send(printCurrentBookings(bot, update.CallbackQuery.Message.Chat.ID))
				case dateTimeMatch.MatchString(update.CallbackQuery.Data):
					date, _ := time.Parse("02/01 (Mon) 3:04 PM", update.CallbackQuery.Data)
					date = date.AddDate(time.Now().Year(), 0, 0)
					toBook = append(toBook, BookingSlot{
						Date : date,
						ChatID:	update.CallbackQuery.Message.Chat.ID,
						Outcome: AWAITING,
					})
					bot.Send(printCurrentBookings(bot, update.CallbackQuery.Message.Chat.ID))
			}
		}
		if update.Message != nil {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

			switch update.Message.Text {
				case "Add":
					year, month, _ = time.Now().Date()
					msg.ReplyMarkup = GenerateCalendar(year, month)
				case "Update":
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "..."))
					updateBookings(client, bot)
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Updated") 
				case "All":
					message := "Current available slots:\n"
					for _, validSlot := range valids { //for all the slots which meet the rule (i.e. within 10 days of now)
						message += validSlot.Date.Format("02/01 (Mon) 3:04 PM") + ": " + validSlot.Availability + "\n"
					}
					msg = tgbotapi.NewMessage(update.Message.Chat.ID, message) 
				case "My":
					msg = printCurrentBookings(bot, update.Message.Chat.ID)
				case "Remove":
					msg.ReplyMarkup = removeSlotKeyboard(update.Message.Chat.ID)
				case "Log":
					log.Println(valids)
			}
			bot.Send(msg)
		}
	}
}

func printCurrentBookings(bot *tgbotapi.BotAPI, ChatID int64) tgbotapi.MessageConfig{
	message := "Your current bookings:\n"
	for _, books := range toBook{
		if books.ChatID == ChatID{
			log.Println(books.Date)
			message += books.Date.Format("02/01 (Mon) 3:04 PM")	+ " " + books.Outcome + "\n"
		}
	}
	return tgbotapi.NewMessage(ChatID, message)
}

func runUpdateBookings(client *http.Client, bot *tgbotapi.BotAPI){
	for{
		updateBookings(client, bot)
		r := 30
		time.Sleep(time.Duration(r) * time.Minute)
	}
}

func updateBookings(client *http.Client, bot *tgbotapi.BotAPI){
	t := time.Now()
	// Start from next day booking
	if t.Hour() >= 18{
		t = t.AddDate(0, 0, 1)
	}
	var temp []ClimbingSlot
	for{
		log.Println("Fetching date " + t.Format("2006-01-02"))
		rawPage := slotPage(client, t.Format("2006-01-02"))
		log.Println("Parsing page")
		slots := extractDates(rawPage)
		temp = append(temp, validSlots(slots)...)
		re := regexp.MustCompile(`NOT AVAILABLE YET`)
		if re.FindString(rawPage) == "NOT AVAILABLE YET"{
			break
		}
		t = t.AddDate(0, 0, 1)
		r := 3
		time.Sleep(time.Duration(r) * time.Second)
	}
	valids = temp
	for _, avaslot := range valids{
		for i, bookSlot := range toBook{
			if(bookSlot.Date == avaslot.Date && bookSlot.Outcome == AWAITING){
				log.Println("Making booking")
				confirmation := bookPage(client, avaslot.GUID, avaslot.Date.Format("2006-01-02"))
				re := regexp.MustCompile(`Your booking is complete!`)
				if(re.FindString(confirmation) != ""){
					toBook[i].Outcome = SUCCESS
					bot.Send(tgbotapi.NewMessage(bookSlot.ChatID, "Booking successful for " + avaslot.Date.Format("02/01 (Mon) 3:04 PM")))
				} else{
					toBook[i].Outcome = FAILED
					bot.Send(tgbotapi.NewMessage(bookSlot.ChatID, "Booking failed for " + avaslot.Date.Format("02/01 (Mon) 3:04 PM")))
				}
			}
		}
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
	func validSlots(slots []ClimbingSlot) []ClimbingSlot {
	valid := make([]ClimbingSlot, 0)

	for _, slot := range slots {
	if slot.Availability != "Full" {
	valid = append(valid, slot)
	}
	}

	return valid
	}

	// Returns the page containing all the slot information
func slotPage(client *http.Client, date string) string {
	reqBody := strings.NewReader("PreventChromeAutocomplete=&random=5efa14b679d1d&iframeid=rgpiframe5efa14b6305ac&mode=e&fctrl_1=offering_guid&offering_guid=7f3e222bae344639836173664e9772ea&fctrl_2=course_guid&course_guid=928713039895fb6db5ed4fbb41a3e3472f0aea5e&fctrl_3=limited_to_course_guid_for_offering_guid_7f3e222bae344639836173664e9772ea&limited_to_course_guid_for_offering_guid_7f3e222bae344639836173664e9772ea=&fctrl_4=show_date&show_date="+date+"&ftagname_0_pcount-pid-1-361000=pcount&ftagval_0_pcount-pid-1-361000=1&ftagname_1_pcount-pid-1-361000=pid&ftagval_1_pcount-pid-1-361000=361000&fctrl_5=pcount-pid-1-361000&pcount-pid-1-361000=0&ftagname_0_pcount-pid-1-361001=pcount&ftagval_0_pcount-pid-1-361001=1&ftagname_1_pcount-pid-1-361001=pid&ftagval_1_pcount-pid-1-361001=361001&fctrl_6=pcount-pid-1-361001&pcount-pid-1-361001=0&ftagname_0_pcount-pid-1-361002=pcount&ftagval_0_pcount-pid-1-361002=1&ftagname_1_pcount-pid-1-361002=pid&ftagval_1_pcount-pid-1-361002=361002&fctrl_7=pcount-pid-1-361002&pcount-pid-1-361002=0&ftagname_0_pcount-pid-1-361003=pcount&ftagval_0_pcount-pid-1-361003=1&ftagname_1_pcount-pid-1-361003=pid&ftagval_1_pcount-pid-1-361003=361003&fctrl_8=pcount-pid-1-361003&pcount-pid-1-361003=0&ftagname_0_pcount-pid-1-361004=pcount&ftagval_0_pcount-pid-1-361004=1&ftagname_1_pcount-pid-1-361004=pid&ftagval_1_pcount-pid-1-361004=361004&fctrl_9=pcount-pid-1-361004&pcount-pid-1-361004=0")
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

var seededRand *rand.Rand = rand.New(
  rand.NewSource(time.Now().UnixNano()))

func StringWithCharset(length int, charset string) string {
  b := make([]byte, length)
  for i := range b {
    b[i] = charset[seededRand.Intn(len(charset))]
  }
  return string(b)
}

func bookPage(client *http.Client, GUID string, date string) string {
	bookingGuid := StringWithCharset(40, charset)
	reqBody := strings.NewReader("PreventChromeAutocomplete=&random=5efad5f3e951f&iframeid=rgpiframe5efaa73c1eccd&mode=e&fctrl_1=booking_guid&booking_guid=" + bookingGuid + "&fctrl_2=customer-firstname&customer-firstname="+os.Getenv("firstname")+"&fctrl_3=customer-lastname&customer-lastname="+os.Getenv("lastname")+"&fctrl_4=customer-middlename&customer-middlename=&fctrl_5=customer-email&customer-email="+os.Getenv("email")+"&fctrl_6=opt-in&fctrl_7=customer-phone&customer-phone="+os.Getenv("phone")+"&fctrl_8=customer-birth-month&customer-birth-month="+os.Getenv("b_month")+"&fctrl_9=customer-birth-day&customer-birth-day="+os.Getenv("b_day")+"&fctrl_10=customer-birth-year&customer-birth-year="+os.Getenv("b_year")+"&profile_password_1=&profile_password_2=&fctrl_11=ptype-english-label-1&ptype-english-label-1=members+%28opwp%2C+prepaid%2C+1+month+multi+gym%29&ftagname_0_pfirstname-pindex-1-1=pfirstname&ftagval_0_pfirstname-pindex-1-1=1&ftagname_1_pfirstname-pindex-1-1=pindex&ftagval_1_pfirstname-pindex-1-1=1&fctrl_12=pfirstname-pindex-1-1&pfirstname-pindex-1-1="+os.Getenv("firstname")+"&ftagname_0_plastname-pindex-1-1=plastname&ftagval_0_plastname-pindex-1-1=1&ftagname_1_plastname-pindex-1-1=pindex&ftagval_1_plastname-pindex-1-1=1&fctrl_13=plastname-pindex-1-1&plastname-pindex-1-1="+os.Getenv("lastname")+"&ftagname_0_pmiddle-pindex-1-1=pmiddle&ftagval_0_pmiddle-pindex-1-1=1&ftagname_1_pmiddle-pindex-1-1=pindex&ftagval_1_pmiddle-pindex-1-1=1&fctrl_14=pmiddle-pindex-1-1&pmiddle-pindex-1-1=&fctrl_15=participant-birth-pindex-1month&participant-birth-pindex-1month="+os.Getenv("b_month")+"&fctrl_16=participant-birth-pindex-1day&participant-birth-pindex-1day="+os.Getenv("b_day")+"&fctrl_17=participant-birth-pindex-1year&participant-birth-pindex-1year="+os.Getenv("b_year")+"&ftagname_0_bookingcc02ab4926a84a61a3454ae088156bd9=isCQ&ftagval_0_bookingcc02ab4926a84a61a3454ae088156bd9=1&ftagname_1_bookingcc02ab4926a84a61a3454ae088156bd9=CQGroup&ftagval_1_bookingcc02ab4926a84a61a3454ae088156bd9=booking&ftagname_2_bookingcc02ab4926a84a61a3454ae088156bd9=CQguid&ftagval_2_bookingcc02ab4926a84a61a3454ae088156bd9=cc02ab4926a84a61a3454ae088156bd9&fctrl_18=bookingcc02ab4926a84a61a3454ae088156bd9&bookingcc02ab4926a84a61a3454ae088156bd9=&ftagname_0_p1e3b93e774a2c44d4ab04966274738d2f=isCQ&ftagval_0_p1e3b93e774a2c44d4ab04966274738d2f=1&ftagname_1_p1e3b93e774a2c44d4ab04966274738d2f=CQGroup&ftagval_1_p1e3b93e774a2c44d4ab04966274738d2f=p1&ftagname_2_p1e3b93e774a2c44d4ab04966274738d2f=CQguid&ftagval_2_p1e3b93e774a2c44d4ab04966274738d2f=e3b93e774a2c44d4ab04966274738d2f&fctrl_19=p1e3b93e774a2c44d4ab04966274738d2f&p1e3b93e774a2c44d4ab04966274738d2f=93530118&fctrl_20=offering_guid&offering_guid=7f3e222bae344639836173664e9772ea&fctrl_21=course_guid&course_guid="+GUID+"&fctrl_22=limited_to_course_guid_for_offering_guid_7f3e222bae344639836173664e9772ea&limited_to_course_guid_for_offering_guid_7f3e222bae344639836173664e9772ea=&fctrl_23=show_date&show_date=+"+date+"&ftagname_0_pcount-pid-1-361000=pcount&ftagval_0_pcount-pid-1-361000=1&ftagname_1_pcount-pid-1-361000=pid&ftagval_1_pcount-pid-1-361000=361000&fctrl_24=pcount-pid-1-361000&pcount-pid-1-361000=1&ftagname_0_pcount-pid-1-361001=pcount&ftagval_0_pcount-pid-1-361001=1&ftagname_1_pcount-pid-1-361001=pid&ftagval_1_pcount-pid-1-361001=361001&fctrl_25=pcount-pid-1-361001&pcount-pid-1-361001=0&ftagname_0_pcount-pid-1-361002=pcount&ftagval_0_pcount-pid-1-361002=1&ftagname_1_pcount-pid-1-361002=pid&ftagval_1_pcount-pid-1-361002=361002&fctrl_26=pcount-pid-1-361002&pcount-pid-1-361002=0&ftagname_0_pcount-pid-1-361003=pcount&ftagval_0_pcount-pid-1-361003=1&ftagname_1_pcount-pid-1-361003=pid&ftagval_1_pcount-pid-1-361003=361003&fctrl_27=pcount-pid-1-361003&pcount-pid-1-361003=0&ftagname_0_pcount-pid-1-361004=pcount&ftagval_0_pcount-pid-1-361004=1&ftagname_1_pcount-pid-1-361004=pid&ftagval_1_pcount-pid-1-361004=361004&fctrl_28=pcount-pid-1-361004&pcount-pid-1-361004=0")
	log.Println(reqBody)
	req, err := http.NewRequest("POST", "https://app.rockgympro.com/b/widget/?a=booking_step3_complete&booking_guid="+bookingGuid, reqBody)
	errCheck(err, "Error querying booking")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := client.Do(req)
	errCheck(err, "Error getting confirmation")
	bytes, err := ioutil.ReadAll(resp.Body)
	errCheck(err, "Error reading confirmation request body")
	// log.Println("Make the book for " + GUID)
	// return "Your booking is complete!
	log.Println(string(bytes))
	return string(bytes)
}

	// Given the output of the slot page, finds the
	func extractDates(slotPage string) []ClimbingSlot {
		daySections := strings.Split(slotPage, "</tr>\n<tr>")[1:]
		slots := make([]ClimbingSlot, 0)
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
			re := regexp.MustCompile(`NOT AVAILABLE YET`)
			if re.FindString(daySection) == "NOT AVAILABLE YET"{
			continue;
			}
			// Get date
			re = regexp.MustCompile(`(\w+, \w+ \w+, \w+:?\w* \w+)`)
			dateString := string(re.FindString(strings.Split(strings.Split(daySection, "</td>")[0], ">")[1]))
			if dateString == ""{
			continue
			}
			// fmt.Println(dateString)
			// Check if its a half hour
			re = regexp.MustCompile(`30|50`)
			var date time.Time
			var err error
			if re.MatchString(dateString){
				date, err = time.Parse("Mon, January 2, 3:04 PM", dateString)
			} else{
				date, err = time.Parse("Mon, January 2, 3 PM", dateString) 
			}
			errCheck(err, "Error parsing date and time of slot")
			date = date.AddDate(time.Now().Year(), 0, 0)
			// Get avability
			re = regexp.MustCompile(`Available|[\d]* spaces?|Full`)
			rawSlots := re.FindString(daySection)
			// Get GUID
			re = regexp.MustCompile(`\w{40}`)
			GUID := re.FindString(daySection)
			// fmt.Println(date)
			// fmt.Println(rawSlots)
			// log.Println(date, rawSlots)
			slots = append(slots, ClimbingSlot{
				Date:          date,
				Availability: rawSlots,
				GUID : GUID,
			})
		}
		return slots
	}

func errCheck(err error, msg string) {
	if err != nil {
	log.Fatal(msg + ": " + err.Error())
	}
}

// GenerateCalendar generates calendar
func GenerateCalendar(year int, month time.Month) tgbotapi.InlineKeyboardMarkup {
	keyboard := tgbotapi.InlineKeyboardMarkup{}
	keyboard = addMonthYearRow(year, month, keyboard)
	keyboard = addDaysNamesRow(keyboard)
	keyboard = generateMonth(year, int(month), keyboard)
	keyboard = addSpecialButtons(keyboard)
	return keyboard
}

// HandlerPrevButton handles prev button
func HandlerPrevButton(year int, month time.Month) (tgbotapi.InlineKeyboardMarkup, int, time.Month) {
	if month != 1 {
		month--
	} else {
		month = 12
		year--
	}
	return GenerateCalendar(year, month), year, month
}

// HandlerNextButton handles next button
func HandlerNextButton(year int, month time.Month) (tgbotapi.InlineKeyboardMarkup, int, time.Month) {
	if month != 12 {
		month++
	} else {
		year++
	}
	return GenerateCalendar(year, month), year, month
}

func addMonthYearRow(year int, month time.Month, keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	var row []tgbotapi.InlineKeyboardButton
	btn := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %v", month, year), "1")
	row = append(row, btn)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)
	return keyboard
}

func addDaysNamesRow(keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	days := [7]string{"Mo", "Tu", "We", "Th", "Fr", "Sa", "Su"}
	var rowDays []tgbotapi.InlineKeyboardButton
	for _, day := range days {
		btn := tgbotapi.NewInlineKeyboardButtonData(day, day)
		rowDays = append(rowDays, btn)
	}
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)
	return keyboard
}

func generateMonth(year int, month int, keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	firstDay := date(year, month, 0)
	amountDaysInMonth := date(year, month + 1, 0).Day()

	weekday := int(firstDay.Weekday())
	rowDays := []tgbotapi.InlineKeyboardButton{}
	for i := 1; i <= weekday; i++ {
		btn := tgbotapi.NewInlineKeyboardButtonData(" ", string(i))
		rowDays = append(rowDays, btn)
	}

	amountWeek := weekday
	for i := 1; i <= amountDaysInMonth; i++ {
		if amountWeek == 7 {
			keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)
			amountWeek = 0
			rowDays = []tgbotapi.InlineKeyboardButton{}
		}

		day := strconv.Itoa(i)
		if len(day) == 1 {
			day = fmt.Sprintf("0%v", day)
		}
		monthStr := strconv.Itoa(month)
		if len(monthStr) == 1 {
			monthStr = fmt.Sprintf("0%v", monthStr)
		}

		btnText := fmt.Sprintf("%v", i)
		if(time.Now().Day() == i) {
			btnText = fmt.Sprintf("%v!", i)
		}
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("%v.%v.%v", year, monthStr, day))
		rowDays = append(rowDays, btn)
		amountWeek++
	}
	for i := 1; i <= 7-amountWeek; i++ {
		btn := tgbotapi.NewInlineKeyboardButtonData(" ", string(i))
		rowDays = append(rowDays, btn)
	}

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)

	return keyboard
}

func date(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func addSpecialButtons(keyboard tgbotapi.InlineKeyboardMarkup) tgbotapi.InlineKeyboardMarkup {
	var rowDays = []tgbotapi.InlineKeyboardButton{}
	btnPrev := tgbotapi.NewInlineKeyboardButtonData(BtnPrev, BtnPrev)
	btnNext := tgbotapi.NewInlineKeyboardButtonData(BtnNext, BtnNext)
	rowDays = append(rowDays, btnPrev, btnNext)
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, rowDays)
	return keyboard
}