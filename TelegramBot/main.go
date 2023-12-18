package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Send any text message to the bot after the bot has been started

// День
type Day struct {
	Name    string
	Lessons []Lesson
}

// Пара
type Lesson struct {
	Name    string
	Teacher string
	Room    string
	Comment string
	Number  int16
	Type    int8
}

type TimeOnly struct {
	Hours   int
	Minutes int
}

type TimeRange struct {
	Start TimeOnly
	End   TimeOnly
}

//
// [ { start: "8:00", end: "9:30" }, { start: "9:50", end: "11:20" } ]

func createTimeOnly(hours, minutes int) TimeOnly {
	return TimeOnly{Hours: hours, Minutes: minutes}
}

var timeTable = []TimeRange{
	{Start: createTimeOnly(8, 00), End: createTimeOnly(9, 50)},
	{Start: createTimeOnly(9, 50), End: createTimeOnly(11, 30)},
	{Start: createTimeOnly(11, 30), End: createTimeOnly(13, 20)},
	{Start: createTimeOnly(13, 20), End: createTimeOnly(15, 00)},
	{Start: createTimeOnly(15, 00), End: createTimeOnly(16, 30)},
}

func convertToMinutes(hours, minutes int) int {
	return hours*60 + minutes
}

func getCurrentLessonNumber(now time.Time) int16 {
	currentLessonNum := -1

	nowMinutes := convertToMinutes(now.Hour(), now.Minute())
	startMinutesFirst := convertToMinutes(timeTable[0].Start.Hours, timeTable[0].Start.Minutes)
	endMinutesLast := convertToMinutes(timeTable[len(timeTable)-1].End.Hours, timeTable[len(timeTable)-1].End.Minutes)

	if nowMinutes < startMinutesFirst {
		return 0
	} else if nowMinutes > endMinutesLast {
		return 8
	}

	for i, timeEntry := range timeTable {
		startMinutes := convertToMinutes(timeEntry.Start.Hours, timeEntry.Start.Minutes)
		endMinutes := convertToMinutes(timeEntry.End.Hours, timeEntry.End.Minutes)

		if nowMinutes >= startMinutes && nowMinutes < endMinutes {
			currentLessonNum = i + 1
			break
		}
	}

	return int16(currentLessonNum)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}

	tokenBuff, err := os.ReadFile("./token.txt")
	if err != nil {
		panic(err)
	}

	b, err := bot.New(string(tokenBuff), opts...)
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, helpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/toadmin", bot.MatchTypeExact, toadminHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/nextLesson", bot.MatchTypeExact, nextLessonHandler)
	// Для дня недели
	b.RegisterHandler(bot.HandlerTypeMessageText, "/scheduleOn", bot.MatchTypeExact, scheduleOnHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/scheduleTomorrow", bot.MatchTypeExact, scheduleTomorrowHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/scheduleToday", bot.MatchTypeExact, scheduleTodayHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/comment", bot.MatchTypeExact, commentHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/whereStudents", bot.MatchTypeExact, whereStudentsHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/whereTeacher", bot.MatchTypeExact, whereTeacherHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/whenExam", bot.MatchTypeExact, whenExamHandler)

	b.Start(ctx)
}

func whenExamHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Период проведения экзаменов:\n" + "15.01.24 - 27.01.23",
	})
}

func whereTeacherHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Команда ещё не реализована",
	})
}

func whereStudentsHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	days, err := WhatWeek()

	if checkForError(ctx, b, update, err) == false {
		return
	}

	now := time.Now()
	weekday := convertWeekdayToNormal(now.Weekday())

	// Определить какая сейчас идёт пара
	currentLessonNumber := getCurrentLessonNumber(now)

	// Пройтись по дню, найти первую пару, которая больше текущего номера
	var Lesson Lesson
	found := false
	for _, lesson := range days[weekday].Lessons {

		if lesson.Number == currentLessonNumber {
			found = true
			Lesson = lesson
		}
	}

	if found == false {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   " У студентов сейчас нет пары :(",
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "В данный момент студенты в " + Lesson.Room,
	})
}

func commentHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Команда ещё не реализована",
	})
}

func Week(now time.Time) int {
	_, thisWeek := now.ISOWeek()
	return thisWeek
}

func WhatWeek() ([]Day, error) {
	week := Week(time.Now().UTC())
	if week%2 != 0 {
		days, err := getDays()
		return days, err
	}
	days, err := getDays2()
	return days, err
}
func scheduleOnHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	days, err := WhatWeek()

	if checkForError(ctx, b, update, err) == false {
		return
	}
	var builder strings.Builder

	for weekday := 0; weekday < 5; weekday++ {
		builder.WriteString(days[weekday].Name)
		builder.WriteString("\n")
		for _, lesson := range days[weekday].Lessons {
			// Добавляем эту строку к stringBuilder
			builder.WriteString(lessonToString(lesson))
			builder.WriteString("\n")
		}
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   builder.String(),
	})

}

func scheduleTomorrowHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	days, err := WhatWeek()

	if checkForError(ctx, b, update, err) == false {
		return
	}

	now := time.Now()
	weekday := convertWeekdayToNormal(now.Weekday()) + 1

	if weekday > 4 && weekday < 6 {
		weekday = 0
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Суббота и воскресенье - выходные дни.\n\n",
		})
	}
	if weekday == 6 {
		weekday = 0
	}
	// Создаём построитель строк (strings.Builder)
	var builder strings.Builder

	builder.WriteString(days[weekday].Name)
	builder.WriteString(":\n")

	// Проходим по всем парам и конвертируем их в строку
	for _, lesson := range days[weekday].Lessons {
		// Добавляем эту строку к stringBuilder
		builder.WriteString(lessonToString(lesson))
		builder.WriteString("\n")
	}

	// Отправляем в телеграм полную строчку
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Расписание на следующий учебный день:\n" + builder.String(),
	})
}

func getDays() ([]Day, error) {
	fileContent, err := os.ReadFile("./json/lessons_odd.json")
	if err != nil {
		return nil, err
	}

	var days []Day
	err = json.Unmarshal(fileContent, &days)
	if err != nil {
		return nil, err
	}

	return days, nil
}

func getDays2() ([]Day, error) {
	fileContent, err := os.ReadFile("./json/lessons_even.json")
	if err != nil {
		return nil, err
	}

	var days []Day
	err = json.Unmarshal(fileContent, &days)
	if err != nil {
		return nil, err
	}

	return days, nil
}
func scheduleTodayHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	days, err := WhatWeek()

	if checkForError(ctx, b, update, err) == false {
		return
	}

	now := time.Now()
	weekday := convertWeekdayToNormal(now.Weekday())

	// Создаём построитель строк (strings.Builder)
	var builder strings.Builder

	builder.WriteString(days[weekday].Name)
	builder.WriteString("\n")

	// Проходим по всем парам и конвертируем их в строку
	for _, lesson := range days[weekday].Lessons {
		// Добавляем эту строку к stringBuilder
		builder.WriteString(lessonToString(lesson))
		builder.WriteString("\n")
	}

	// Отправляем в телеграм полную строчку
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   builder.String(),
	})
}

func lessonToString(lesson Lesson) string {
	var lessonType string
	if lesson.Type == 0 {
		lessonType = "Лекция"
	} else if lesson.Type == 1 {
		lessonType = "Практика"
	} else {
		lessonType = "Неизвестный тип"
	}

	return strconv.Itoa(int(lesson.Number)) + ")" + "'" + lesson.Name + "' " + "\n" + "- " + lessonType + "\n" + "- " + lesson.Teacher + "\n" + " (" + lesson.Room + ")" + "\n"
}

func convertWeekdayToNormal(weekday time.Weekday) int8 {
	var temp = int8(weekday - 1)
	// Воскресенье
	if temp == -1 {
		return 6
	}

	return temp
}

func checkForError(ctx context.Context, b *bot.Bot, update *models.Update, err error) bool {
	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Произошла ошибка: " + err.Error(),
		})
		return false
	}

	return true
}

func nextLessonHandler(ctx context.Context, b *bot.Bot, update *models.Update) {

	days, err := WhatWeek()

	if checkForError(ctx, b, update, err) == false {
		return
	}

	now := time.Now()
	weekday := convertWeekdayToNormal(now.Weekday())

	// Определить какая сейчас идёт пара
	currentLessonNumber := getCurrentLessonNumber(now)

	// Пройтись по дню, найти первую пару, которая больше текущего номера
	var nextLesson Lesson
	found := false
	for _, lesson := range days[weekday].Lessons {
		if lesson.Number > currentLessonNumber {
			nextLesson = lesson
			found = true
			break
		}
	}

	if found == false {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Текущая пара: " + strconv.Itoa(int(currentLessonNumber)) + "\nНе смог найти следующую пару :(",
		})
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Текущая пара: " + strconv.Itoa(int(currentLessonNumber)) + "\nСледующая, пара:\n" + lessonToString(nextLesson),
	})
}

func toadminHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Команда ещё не реализована",
	})
}

func helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	commands := []string{
		"/help",
		"/toadmin",
		"/nextLesson",
		"/scheduleOn",
		"/scheduleTomorrow",
		"/scheduleToday",
		"/comment",
		"/whereStudents",
		"/whereTeacher",
		"/whenExam",
	}
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Доступные команды:\n" + strings.Join(commands, ", "),
	})
}
func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Используйте /help для списка команд",
	})
}
