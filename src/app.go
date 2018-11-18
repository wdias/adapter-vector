package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/kataras/iris"
	"github.com/kataras/iris/context"
)

const (
	influxdbURL = "http://adapter-scalar-influxdb.default.svc.cluster.local:8086"
	database    = "wdias"
	username    = "wdias"
	password    = "wdias123"
)

type Points []struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
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

func writePoints(clnt client.Client) {
	sampleSize := 1000

	bp, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: "s",
	})
	if err != nil {
		log.Fatal(err)
	}

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < sampleSize; i++ {
		regions := []string{"us-west1", "us-west2", "us-west3", "us-east1"}
		tags := map[string]string{
			"cpu":    "cpu-total",
			"host":   fmt.Sprintf("host%d", rand.Intn(1000)),
			"region": regions[rand.Intn(len(regions))],
		}

		idle := rand.Float64() * 100.0
		fields := map[string]interface{}{
			"idle": idle,
			"busy": 100.0 - idle,
		}

		pt, err := client.NewPoint(
			"cpu_usage",
			tags,
			fields,
			time.Now(),
		)
		if err != nil {
			log.Fatal(err)
		}
		bp.AddPoint(pt)
	}

	if err := clnt.Write(bp); err != nil {
		log.Fatal(err)
	}
}

func main() {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxdbURL,
		// Username: username,
		// Password: password,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	app := iris.Default()

	app.Post("/timeseries", func(ctx iris.Context) {
		points := &Points{}
		err := ctx.ReadJSON(points)
		if err != nil {
			ctx.JSON(context.Map{"response": err.Error()})
		} else {
			fmt.Println("Successfully inserted into database")
			ctx.JSON(context.Map{"response": "User succesfully created", "message": points})
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
