package main

import (
"encoding/json"
"fmt"
"net/http"
"github.com/julienschmidt/httprouter"
"gopkg.in/mgo.v2"
"strings"
"gopkg.in/mgo.v2/bson"
"io/ioutil"
"strconv"
)


// List of price estimates
type PriceEstimates struct {
	Prices         []PriceEstimate `json:"prices"`
}

// Uber price estimate
type PriceEstimate struct {
	ProductId       string  `json:"product_id"`
	CurrencyCode    string  `json:"currency_code"`
	DisplayName     string  `json:"display_name"`
	Estimate        string  `json:"estimate"`
	LowEstimate     int     `json:"low_estimate"`
	HighEstimate    int     `json:"high_estimate"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Duration        int     `json:"duration"`
	Distance        float64 `json:"distance"`
}

type UberOutput struct{
	Cost int
	Duration int
	Distance float64
}

type InputAddress struct {
		Name   string        `json:"name"`
		Address string 		`json:"address"`
		City string			`json:"city"`
		State string		`json:"state"`
		Zip string			`json:"zip"`
	}



type OutputAddress struct {

		Id     bson.ObjectId `json:"_id" bson:"_id,omitempty"`
		Name   string        `json:"name"`
		Address string 		`json:"address"`
		City string			`json:"city" `
		State string		`json:"state"`
		Zip string			`json:"zip"`

		Coordinate struct{
			Lat string 		`json:"lat"`
			Lang string 	`json:"lang"`
		}
	}



type GoogleCoordinates struct {
	Results []struct {
		AddressComponents []struct {
			LongName  string   `json:"long_name"`
			ShortName string   `json:"short_name"`
			Types     []string `json:"types"`
		} `json:"address_components"`
		FormattedAddress string `json:"formatted_address"`
		Geometry struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
			LocationType string `json:"location_type"`
			Viewport     struct {
				Northeast struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"northeast"`
				Southwest struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"southwest"`
			} `json:"viewport"`
		} `json:"geometry"`
		PlaceID string   `json:"place_id"`
		Types   []string `json:"types"`
	} `json:"results"`
	Status string `json:"status"`
}

type Response struct {

      Id bson.ObjectId `json:"id" bson:"_id"`
      Name string `json:"name" bson:"name"`
      Address string `json:"address" bson:"address"`
      City string `json:"city" bson:"city"`
      State string `json:"state" bson:"state"`
      Zip string `json:"zip" bson:"zip"`
      Coordinate struct 
	  {
	   Lat string `json:"lat"   bson:"lat"`
	   Lng string `json:"lng"   bson:"lng"`		
	  }`json:"coordinate" bson:"coordinate"`
}


type TripPostInput struct{
	Starting_from_location_id   string    `json:"starting_from_location_id"`
	Location_ids []string
}

type TripPostOutput struct{
	Id     bson.ObjectId 				  `json:"_id" bson:"_id,omitempty"`
	Status string  						  `json:"status"`
	Starting_from_location_id   string    `json:"starting_from_location_id"`
	Best_route_location_ids []string
	Total_uber_costs int			  `json:"total_uber_costs"`
	Total_uber_duration int			  `json:"total_uber_duration"`
	Total_distance float64				  `json:"total_distance"`

}		

type MongoSession struct {
				session *mgo.Session
			}

			
func newMongoSession(session *mgo.Session) *MongoSession {
	return &MongoSession{session}
}

func (ms MongoSession) GetLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	
	id := params.ByName("id")
    if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}
	
	fmt.Print("Before OID")
	oid := bson.ObjectIdHex(id)
	fmt.Print("OID is", oid)

	resp := Response{}
	
	if err := ms.session.DB("cmpe273").C("locations").FindId(oid).One(&resp); err != nil {
		fmt.Print("Inside fail case")
		w.WriteHeader(404)
		return
	}

	json.NewDecoder(r.Body).Decode(resp)

	mObject, _ := json.Marshal(resp)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", mObject)
}



func (ms MongoSession) CreateLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {


	resp := Response{}

	json.NewDecoder(r.Body).Decode(&resp)

	data := callGoogleAPI(&resp)
	
	data.Id = bson.NewObjectId()

	ms.session.DB("cmpe273").C("locations").Insert(data)
	
	mObject, _ := json.Marshal(data)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", mObject)
}


func (ms MongoSession) DeleteLocation(w http.ResponseWriter, r *http.Request, params httprouter.Params) {

	    
	    id := params.ByName("id")

	   
	    if !bson.IsObjectIdHex(id) {
	        w.WriteHeader(404)
	        return
	    }

	    oid := bson.ObjectIdHex(id)
	    if err := ms.session.DB("cmpe273").C("locations").RemoveId(oid); err != nil {
		    fmt.Print("Inside fail case")
	        w.WriteHeader(404)
	        return
	    }
	   
	    w.WriteHeader(200)
}


func (ms MongoSession) UpdateLocation (w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	
	id := params.ByName("id")

	if !bson.IsObjectIdHex(id) {
		w.WriteHeader(404)
		return
	}

	oid := bson.ObjectIdHex(id)

	get := Response{}
	put := Response{}

	put.Id = oid

	json.NewDecoder(r.Body).Decode(&put)

	if err := ms.session.DB("cmpe273").C("locations").FindId(oid).One(&get); err != nil {
		w.WriteHeader(404)
		return
	}

	na := get.Name

	object := ms.session.DB("cmpe273").C("locations")

	get = callGoogleAPI(&put)
	object.Update(bson.M{"_id": oid}, bson.M{"$set": bson.M{ "address": put.Address, "city": put.City, "state": put.State, "zip" : put.Zip, "coordinate": bson.M{"lat" : get.Coordinate.Lat, "lng" : get.Coordinate.Lng}}})

	get.Name = na

	mObject, _ := json.Marshal(get)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", mObject)

}

func callGoogleAPI (resp *Response) Response {

  address := resp.Address
  city := resp.City

  gstate := strings.Replace(resp.State," ","+",-1)
  gaddress := strings.Replace(address, " ", "+", -1)
  gcity := strings.Replace(city," ","+",-1)

	uri := "http://maps.google.com/maps/api/geocode/json?address="+gaddress+"+"+gcity+"+"+gstate+"&sensor=false"


    result, _ := http.Get(uri)

	body, _ := ioutil.ReadAll(result.Body)


 	Cords := GoogleCoordinates{}

    err := json.Unmarshal(body, &Cords)
    if err!= nil {
      panic(err)
    } 


	 for _, Sample := range Cords.Results {
				resp.Coordinate.Lat= strconv.FormatFloat(Sample.Geometry.Location.Lat, 'f', 7, 64)
				resp.Coordinate.Lng = strconv.FormatFloat(Sample.Geometry.Location.Lng, 'f', 7, 64)
		}

   return *resp
}


func (ms MongoSession) CreateTrip(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var tI TripPostInput
	var tO TripPostOutput
	// var cost_map map[string]int
	var cost_array []int
	var duration_array []int
	var distance_array []float64
	cost_total := 0
	duration_total := 0
	distance_total := 0.0
	
	json.NewDecoder(r.Body).Decode(&tI)	

	starting_id:= bson.ObjectIdHex(tI.Starting_from_location_id)
	fmt.Println("Starting ID is" ,starting_id)
	var start Response
	if err := ms.session.DB("cmpe273").C("locations").FindId(starting_id).One(&start); err != nil {
       	w.WriteHeader(404)
        return
    }
	
	name := start.Name
	fmt.Println("Name is", name)
    start_Lat := start.Coordinate.Lat
    start_Lang := start.Coordinate.Lng
	fmt.Println("Start here", start_Lat)
    Location_ids := tI.Location_ids
	fmt.Println("LocationIDS are" , Location_ids)

    
			
			for _, loc := range tI.Location_ids{
				// var cost_array []int
				id := bson.ObjectIdHex(loc)
				fmt.Println("Id is ",loc)
				fmt.Println("ZZZZZZZZz is", id)
				var o Response
				if err := ms.session.DB("cmpe273").C("locations").FindId(id).One(&o); err != nil {
		       		w.WriteHeader(404)
		        	return
		    	}
		    	loc_Lat := o.Coordinate.Lat
		    	loc_Lang := o.Coordinate.Lng
		    	
				fmt.Println("start lat", start_Lat)
				fmt.Println("start lon", start_Lang)
				fmt.Println("End lat", loc_Lat)
				fmt.Println("End Lon string" , loc_Lang)
				
		    	getUberResponse := Get_uber_price(start_Lat, start_Lang, loc_Lat, loc_Lang)
		    	fmt.Println("Uber Response is: ", getUberResponse.Cost, getUberResponse.Duration, getUberResponse.Distance );
				
		    	cost_array = append(cost_array, getUberResponse.Cost)
		    	duration_array = append(duration_array, getUberResponse.Duration)
		    	distance_array = append(distance_array, getUberResponse.Distance)
		    	
			}
			fmt.Println("Cost Array", cost_array)

			min_cost:= cost_array[0]
			var indexNeeded int
			for index, value := range cost_array {
		        if value < min_cost {
		            min_cost = value // found another smaller value, replace previous value in min
		            indexNeeded = index
		        }
		    }
			// fmt.Println("Min Cost", min_cost)
			// // fmt.Println(indexNeeded)
			// // fmt.Println(tI.Location_ids[indexNeeded])
			// fmt.Println("Best", tO.Best_route_location_ids)

			cost_total += min_cost
			duration_total += duration_array[indexNeeded]
			distance_total += distance_array[indexNeeded]

			tO.Best_route_location_ids = append(tO.Best_route_location_ids, tI.Location_ids[indexNeeded])
			// fmt.Println("Best", tO.Best_route_location_ids)

			starting_id = bson.ObjectIdHex(tI.Location_ids[indexNeeded])
			if err := ms.session.DB("cmpe273").C("locations").FindId(starting_id).One(&start); err != nil {
       			w.WriteHeader(404)
        		return
    		}
    		tI.Location_ids = append(tI.Location_ids[:indexNeeded], tI.Location_ids[indexNeeded+1:]...)
			// fmt.Println("Af Location ids", tI.Location_ids)

    		start_Lat = start.Coordinate.Lat
    		start_Lang = start.Coordinate.Lng

    		// Re-initializing the arrays------
    		cost_array = cost_array[:0]
    		duration_array = duration_array[:0]
    		distance_array = distance_array[:0]
    		// fmt.Println("Cost Array", cost_array)

	


	Last_loc_id := bson.ObjectIdHex(tO.Best_route_location_ids[len(tO.Best_route_location_ids)-1])
	var o2 Response
	if err := ms.session.DB("cmpe273").C("locations").FindId(Last_loc_id).One(&o2); err != nil {
		w.WriteHeader(404)
		return
	}
	last_loc_Lat := o2.Coordinate.Lat
	last_loc_Lang := o2.Coordinate.Lng

	ending_id:= bson.ObjectIdHex(tI.Starting_from_location_id)
	var end Response
	if err := ms.session.DB("cmpe273").C("locations").FindId(ending_id).One(&end); err != nil {
       	w.WriteHeader(404)
        return
    }
    end_Lat := end.Coordinate.Lat
    end_Lang := end.Coordinate.Lng
		    	
	getUberResponse_last := Get_uber_price(last_loc_Lat, last_loc_Lang, end_Lat, end_Lang)


	tO.Id = bson.NewObjectId()
	tO.Status = "planning"
	tO.Starting_from_location_id = tI.Starting_from_location_id
	tO.Total_uber_costs = cost_total + getUberResponse_last.Cost
	tO.Total_distance = distance_total + getUberResponse_last.Distance
	tO.Total_uber_duration = duration_total + getUberResponse_last.Duration
	

	// Write the user to mongo
	ms.session.DB("cmpe273").C("Trips").Insert(tO)

	// Marshal provided interface into JSON structure
	uj, _ := json.Marshal(tO)
	// Write content-type, statuscode, payload
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	fmt.Fprintf(w, "%s", uj)
}


func Get_uber_price(startLat, startLon, endLat, endLon string) UberOutput{
	client := &http.Client{}
	
	fmt.Println("start lat", startLat)
	fmt.Println("start lon", startLon)
	fmt.Println("End lat", endLat)
	fmt.Println("End Lon string" , endLon)
	
	reqURL := fmt.Sprintf("https://sandbox-api.uber.com/v1/estimates/price?start_latitude=%s&start_longitude=%s&end_latitude=%s&end_longitude=%s", startLat, startLon, endLat, endLon)
	fmt.Println("URL formed: "+ reqURL)
	
	// res, err := http.GET(reqURL,)
	req, err := http.NewRequest("GET", reqURL , nil)
	req.Header.Set("Authorization", "Token JaiBIl-1gB5pqBkMa0dgPANW4e5MqJ_1AyoTQ0AV")
	
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("error in sending req to Uber: ", err);	
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("error in reading response: ", err);	
	}

	var res PriceEstimates
	err = json.Unmarshal(body, &res)
	if err != nil {
		fmt.Println("error in unmashalling response: ", err);	
	}
    fmt.Println("ARRAYYYYYY FTW" , res)
	var uberOutput UberOutput
	uberOutput.Cost = res.Prices[0].LowEstimate
	uberOutput.Duration = res.Prices[0].Duration
	uberOutput.Distance = res.Prices[0].Distance

	return uberOutput

}





func getConnection() *mgo.Session {

    conn, err := mgo.Dial("mongodb://hemsam:hemsam@ds029803.mongolab.com:29803/cmpe273")

    if err != nil {
        panic(err)
    }
    return conn
}

func main() {

    r := httprouter.New()
 
  	ms := newMongoSession(getConnection())
	
  	r.GET("/locations/:id", ms.GetLocation)
  	r.POST("/locations",ms.CreateLocation)
	r.DELETE("/locations/:id",ms.DeleteLocation)
	r.PUT("/locations/:id", ms.UpdateLocation)
	
	r.POST("/trips", ms.CreateTrip)
	
	http.ListenAndServe("localhost:8080",r)

}
