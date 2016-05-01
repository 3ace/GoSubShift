package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "os"
    "regexp"
    "strings"
    "time"
    "path"
)

type SubData struct {
    Num         string
    TimeStart   time.Time
    TimeEnd     time.Time
    Text        string
}

const TIME_FORMAT    = "15:04:05.000"

var timeReference   time.Time
var subLines        []SubData
var replaceComma    bool
var subFileName     string

func main() {
    subFile := flag.String("f", "", "subtitle file")
    shiftTime := flag.String("t", "", "time shift with format hh:mm:ss.000")

    replaceComma = false

    flag.Parse()

    log.SetFlags(0)

    timeReference = setClock(time.Now(), 0, 0, 0, 0)

    readSubFile(*subFile)
    shiftSub(*shiftTime)
    writeSub()
}

func readSubFile(fileName string) {
    subFileName = fileName

    log.Println("Processing " + subFileName)

    file, err := os.Open(subFileName)

    if err != nil {
        log.Fatalf("Error opening subtitle file: %v", err)
        return
    }

    defer file.Close()

    scanner := bufio.NewScanner(file)

    subLines = make([]SubData, 0)

    var sub SubData

    for scanner.Scan() {
        text := scanner.Text()

        reNum := regexp.MustCompile(`^\d+$`)
        reTime := regexp.MustCompile(`^(\d{2}:\d{2}:[\d(,|\.)]+).+(\d{2}:\d{2}:[\d(,|\.)]+)$`)

        switch {
        case reNum.MatchString(text):
            sub = SubData{}
            sub.Num = text

        case reTime.MatchString(text):
            subtime := reTime.FindStringSubmatch(text)

            if strings.LastIndex(subtime[1], ",") != -1 {
                subtime[1] = strings.Replace(subtime[1], ",", ".", 1)
                subtime[2] = strings.Replace(subtime[2], ",", ".", 1)

                replaceComma = true
            }

            timeStart, _ := time.Parse(TIME_FORMAT, subtime[1])
            timeEnd, _ := time.Parse(TIME_FORMAT, subtime[2])

            sub.TimeStart = setClock(timeReference,
                timeStart.Hour(), timeStart.Minute(),
                timeStart.Second(), timeStart.Nanosecond())
            sub.TimeEnd = setClock(timeReference,
                timeEnd.Hour(), timeEnd.Minute(),
                timeEnd.Second(), timeEnd.Nanosecond())

        case text != "":
            sub.Text += text + "\n"

        default:
            sub.Text = strings.TrimSpace(sub.Text)
            subLines = append(subLines, sub)

        }
    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
}

func shiftSub(shift string) {
    timeFormat := []string{
        "5",
        "4:5",
        "15:4:5",
        "5.000",
        "4:5.000",
        "15:4:5.000",
    }

    var sTime time.Time
    var err error
    sign := 1;
    trimmedShift := shift

    if strings.HasPrefix(shift, "-") {
        trimmedShift = strings.Trim(shift, "- ")
        sign = -1;
    }

    for i := 0; i < len(timeFormat); i++ {
        sTime, err = time.Parse(timeFormat[i], trimmedShift)

        if err != nil {
            if i < 2 {
                continue
            } else {
                log.Fatalf("shift time (%s) not recognized", shift)
            }
        }

        break;
    }

    tfmt := sTime.Format(fmt.Sprintf("%d:04:05.000", sign * 15))
    log.Println("Shifting by: " + tfmt)

    shiftDur := time.Duration(sTime.Hour()) * time.Hour +
                time.Duration(sTime.Minute()) * time.Minute +
                time.Duration(sTime.Second()) * time.Second +
                time.Duration(sTime.Nanosecond()) * time.Nanosecond

    if sign == -1 {
        shiftDur = -shiftDur
    }

    for i, sub := range subLines {
        var timeStart, timeEnd time.Time

        timeStart = sub.TimeStart.Add(shiftDur)
        timeEnd = sub.TimeEnd.Add(shiftDur)

        if timeStart.Before(timeReference) {
            timeStart = setClock(timeReference, 0, 0, 0, 0)
        }

        if timeEnd.Before(timeReference) {
            timeEnd = setClock(timeReference, 0, 0, 0, 0)
        }

        sub.TimeStart = timeStart
        sub.TimeEnd = timeEnd

        subLines[i] = sub
    }
}

func writeSub() {
    ext := path.Ext(subFileName)
    fname := strings.TrimSuffix(subFileName, ext)
    resName := fname + "-resync" + ext

    file, err := os.Create(resName)

    if err != nil {
        log.Fatalf("Error creating new subtitle file: %v", err)
        return
    }

    defer file.Close()

    for _, sub := range subLines {
        file.WriteString(sub.Num + "\n")

        start := sub.TimeStart.Format("15:04:05.000")
        end := sub.TimeEnd.Format("15:04:05.000")

        file.WriteString(start + " --> " + end + "\n")
        file.WriteString(sub.Text + "\n\n")
    }

    log.Printf("Result has been saved as %s", resName)
}

func setClock(originalTime time.Time, hour, minute, second, nanosecond int) time.Time {
    refYear, refMonth, refDay := originalTime.Date()

    return time.Date(refYear, refMonth, refDay,
        hour, minute, second, nanosecond,
        originalTime.Location())
}
