package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli"
)

type Candle struct {
	High float64
	Low  float64
}

var flags = []cli.Flag{
	cli.Int64Flag{
		EnvVar: "START",
		Name:   "start",
	},
	cli.Int64Flag{
		EnvVar: "END",
		Name:   "end",
	},
	cli.StringFlag{
		EnvVar: "SYMBOL",
		Name:   "symbol",
	},
}

const URL = "https://testnet-dex.binance.org"

func volatility(c *cli.Context) {
	vol, err := Volatility(c.Int64("start"), c.Int64("end"), c.String("symbol"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("volatility score: %f with bonus: %f\n", vol, 3*vol)
}

func Volatility(s, e int64, symbol string) (float64, error) {
	market := strings.ToUpper(symbol) + "_BNB"
	start := time.Unix(0, s*1000000)
	end := time.Unix(0, e*1000000)
	fmt.Printf("%s\n", market)
	fmt.Printf("%s to %s\n", start, end)

	candles := FetchCandles(start, end, market)

	sum := 0.0
	for _, c := range candles {
		vol := c.High - c.Low
		if c.High <= 15 {
			if vol > 10 {
				vol = 10.0
			}
			sum += vol
		}
	}
	return sum, nil
}

func FetchCandles(start time.Time, end time.Time, market string) []Candle {
	shift := time.Hour * 24
	candles := []Candle{}
	for start.UnixNano() < end.UnixNano() {
		params := url.Values{}
		params.Set("symbol", market)
		params.Set("interval", "5m")
		params.Set("startTime", strconv.FormatInt(start.UnixNano()/1000000, 10))
		params.Set("endTime", strconv.FormatInt(start.Add(shift).UnixNano()/1000000, 10))
		params.Set("limit", "1000")

		var cs [][]interface{}
		err := SendHTTPGetRequest(fmt.Sprintf("%s/%s?%s", URL, "api/v1/klines", params.Encode()), true, false, &cs)
		if err != nil {
			panic(err)
		}

		for _, c := range cs {
			high, err := strconv.ParseFloat(c[2].(string), 64) // index 2 is high
			if err != nil {
				panic(err)
			}
			low, err := strconv.ParseFloat(c[3].(string), 64) // index 3 is low
			if err != nil {
				panic(err)
			}
			candles = append(candles, Candle{High: high, Low: low})
		}
		start = start.Add(shift)
	}
	return candles
}

func SendHTTPGetRequest(url string, jsonDecode, isVerbose bool, result interface{}) error {
	if isVerbose {
		log.Println("Raw URL: ", url)
	}

	res, err := http.Get(url)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("HTTP status code %d", res.StatusCode)
	}

	contents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if isVerbose {
		log.Println("Raw Resp: ", string(contents[:]))
	}

	defer res.Body.Close()

	if jsonDecode {
		err := json.Unmarshal(contents, result)
		if err != nil {
			return err
		}
	}

	return nil
}
