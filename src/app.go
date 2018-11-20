package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

const (
	influxdbURL   = "http://adapter-scalar-influxdb.default.svc.cluster.local:8086"
	database      = "wdias"
	username      = "wdias"
	password      = "wdias123"
	adapterScalar = "http://adapter-metadata.default.svc.cluster.local"
)

type points []struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

type timeseries struct {
	TimeseriesID   string `json:"timeseriesId"`
	ModuleID       string `json:"moduleId"`
	ValueType      string `json:"valueType"`
	ParameterID    string `json:"parameterId"`
	LocationID     string `json:"locationId"`
	TimeseriesType string `json:"timeseriesType"`
	TimeStepID     string `json:"timeStepId"`
}

func queryDB(clnt client.Client, cmd string) (res []client.Result, err error) {
	q := client.Query{
		Command:  cmd,
		Database: database,
	}
	if response, err := clnt.Query(q); err == nil {
		if response.Error() != nil {
			return res, response.Error()
		}
		res = response.Results
	} else {
		return res, err
	}
	return res, nil
}

func writePoints(clnt client.Client, timeseries timeseries, dataPoints *points) (err error) {
	fmt.Println("writePoints:", timeseries, dataPoints)
	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: "s",
	})
	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, point := range *dataPoints {
		tags := map[string]string{
			"timeseriesId":   timeseries.TimeseriesID,
			"moduleId":       timeseries.ModuleID,
			"valueType":      timeseries.ValueType,
			"parameterId":    timeseries.ParameterID,
			"locationId":     timeseries.LocationID,
			"timeseriesType": timeseries.TimeseriesType,
			"timeStepId":     timeseries.TimeStepID,
		}
		fields := map[string]interface{}{
			"value": point.Value,
		}
		t, err := time.Parse(time.RFC3339, point.Time)
		if err != nil {
			fmt.Println("Error: Parsing time with:", point.Time, err)
			continue
		}
		pt, err := client.NewPoint(
			timeseries.TimeseriesType, // measurement
			tags,
			fields,
			t,
		)
		if err != nil {
			fmt.Println(err)
			continue
		}
		bp.AddPoint(pt)
	}

	fmt.Println("wirting to influx")
	if err := clnt.Write(bp); err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

var tr = &http.Transport{
	MaxIdleConns:       10,
	IdleConnTimeout:    30 * time.Second,
	DisableCompression: true,
	Dial: (&net.Dialer{
		Timeout: 5 * time.Second,
	}).Dial,
	TLSHandshakeTimeout: 5 * time.Second,
}
var netClient = &http.Client{
	Transport: tr,
	Timeout:   time.Second * 10,
}

func main() {
	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxdbURL,
		// Username: username,
		// Password: password,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer influxClient.Close()
	q := client.Query{
		Command:  fmt.Sprintf("CREATE DATABASE %s", database),
		Database: database,
	}
	if response, err := influxClient.Query(q); err == nil {
		if response.Error() != nil {
			log.Fatal(response.Error())
		}
		fmt.Println("Connected to the database:", database)
	}

	app := iris.Default()

	app.Post("/timeseries/{timeseriesID:string}", func(ctx iris.Context) {
		timeseriesID := ctx.Params().Get("timeseriesID")
		dataPoints := &points{}
		err := ctx.ReadJSON(dataPoints)
		if err != nil {
			ctx.JSON(context.Map{"response": err.Error()})
		} else {
			fmt.Println("timeseriesID:", timeseriesID, dataPoints)
			fmt.Println("URL:", fmt.Sprint(adapterScalar, "/timeseries/", timeseriesID))
			response, err := netClient.Get(fmt.Sprint(adapterScalar, "/timeseries/", timeseriesID))
			if err != nil {
				ctx.JSON(context.Map{"response": err.Error()})
			}
			defer response.Body.Close()
			body, err := ioutil.ReadAll(response.Body)
			if err != nil {
				ctx.JSON(context.Map{"response": err.Error()})
			}
			var data timeseries
			err = json.Unmarshal(body, &data)
			if err != nil {
				ctx.JSON(context.Map{"response": err.Error()})
			}
			if err := writePoints(influxClient, data, dataPoints); err != nil {
				ctx.JSON(context.Map{"response": err.Error()})
			}
			fmt.Println("Stored timeseries:", data)
			ctx.JSON(context.Map{"response": "Stored data points", "timeseries": data})
		}
	})

	app.Get("/hc", func(ctx iris.Context) {
		ctx.JSON(iris.Map{
			"message": "OK",
		})
	})
	// listen and serve on http://0.0.0.0:8080.
	app.Run(iris.Addr(":8080"))
}
