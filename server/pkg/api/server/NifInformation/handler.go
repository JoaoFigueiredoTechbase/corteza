package nifinformation

import (
	"encoding/json"
	"log"
	"net/http"
)

/*
{
    "result": "success",
    "records": {
        "513602011": {
            "nif": 513602011,
            "seo_url": "mendes-l-it-communications-unipessoal-lda",
            "title": "Mendes L. It & Communications, Unipessoal Lda",
            "address": "Rua Padre Francisco Rodrigues, Nº 2250",
            "pc4": "4800",
            "pc3": "606",
            "city": "Prazins Santa Eufémia",
            "start_date": "2015-06-30",
            "activity": "Actividades de serviços de consultoria e formação em engenharia de comunicações e informática. Comércio a retalho e por grosso de equipamentos de comunicações e informática. Implementação de redes de dados e informática.",
            "status": "active",
            "cae": [
                "62020",
                "47420",
                "47410"
            ],
            "contacts": {
                "email": "geral@techbase.pt",
                "phone": "220035908",
                "website": "www.techbase.pt",
                "fax": "220035908"
            },
            "structure": {
                "nature": "UNI",
                "capital": "5000.00",
                "capital_currency": "EUR"
            },
            "geo": {
                "region": "Braga",
                "county": "Guimarães",
                "parish": "Santa Eufémia Prazins"
            },
            "place": {
                "address": "Rua Padre Francisco Rodrigues, Nº 2250",
                "pc4": "4800",
                "pc3": "606",
                "city": "Prazins Santa Eufémia"
            },
            "racius": "https://www.racius.com/mendes-l-it-communications-unipessoal-lda/",
            "portugalio": "https://www.portugalio.com/mendes-l-it-communications/"
        }
    },
    "nif_validation": false,
    "is_nif": false,
    "credits": {
        "used": "free",
        "left": {
            "month": 994,
            "day": 94,
            "hour": 6,
            "minute": 0,
            "paid": 0
        }
    }
}
*/

type Request struct {
	Request string `json:"request"`
}

type ClientInformation struct {
	ClientName string `json:"client_name"`
	ClientNif  string `json:"client_nif"`
	ApiKey     string `json:"api_key"`
}

type ApiResponse struct {
	Result  string                     `json:"result"`
	Records map[string]json.RawMessage `json:"records"`
}

type NifApiResponse struct {
	Nif        int      `json:"nif"`
	Title      string   `json:"title"`
	Address    string   `json:"address"`
	Pc4        string   `json:"pc4"`
	Pc3        string   `json:"pc3"`
	City       string   `json:"city"`
	Activity   string   `json:"activity"`
	CaeList    []string `json:"cae"`
	Email      string   `json:"email"`
	Phone      string   `json:"phone"`
	Website    string   `json:"website"`
	Fax        string   `json:"fax"`
	Region     string   `json:"region"`
	County     string   `json:"county"`
	Parish     string   `json:"parish"`
	RaciusLink string   `json:"racius"`
}

func HandleClientInformationSearch(w http.ResponseWriter, r *http.Request) {
	log.Println("Received Client Information Search Request")

	var payload ClientInformation
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("ERROR: Failed to decode request body: %v\n", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var client_name = payload.ClientName
	var client_nif = payload.ClientNif
	var api_key = payload.ApiKey

	if len(client_nif) != 0 && len(client_nif) == 9 {
		var HttpRequestUrl = "http://www.nif.pt/?json=1&q=" + client_nif + "&key=" + api_key

		resp, err := http.Get(HttpRequestUrl)
		if err != nil {
			log.Printf("ERROR: Failed to make request %s: %v\n", HttpRequestUrl, err)
			http.Error(w, "failed to make request", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var apiResp ApiResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			log.Printf("ERROR: Failed to decode JSON: %v\n", err)
			http.Error(w, "failed to decode response", http.StatusInternalServerError)
			return
		}

		for _, raw := range apiResp.Records {
			// intermediate struct to flatten "contacts" and "geo"
			var tmp struct {
				Nif      int      `json:"nif"`
				Title    string   `json:"title"`
				Address  string   `json:"address"`
				Pc4      string   `json:"pc4"`
				Pc3      string   `json:"pc3"`
				City     string   `json:"city"`
				Activity string   `json:"activity"`
				CaeList  []string `json:"cae"`
				Contacts struct {
					Email   string `json:"email"`
					Phone   string `json:"phone"`
					Website string `json:"website"`
					Fax     string `json:"fax"`
				} `json:"contacts"`
				Geo struct {
					Region string `json:"region"`
					County string `json:"county"`
					Parish string `json:"parish"`
				} `json:"geo"`
				Racius string `json:"racius"`
			}

			if err := json.Unmarshal(raw, &tmp); err != nil {
				panic(err)
			}

			// map into your target struct
			record := NifApiResponse{
				Nif:        tmp.Nif,
				Title:      tmp.Title,
				Address:    tmp.Address,
				Pc4:        tmp.Pc4,
				Pc3:        tmp.Pc3,
				City:       tmp.City,
				Activity:   tmp.Activity,
				CaeList:    tmp.CaeList,
				Email:      tmp.Contacts.Email,
				Phone:      tmp.Contacts.Phone,
				Website:    tmp.Contacts.Website,
				Fax:        tmp.Contacts.Fax,
				Region:     tmp.Geo.Region,
				County:     tmp.Geo.County,
				Parish:     tmp.Geo.Parish,
				RaciusLink: tmp.Racius,
			}

			// fmt.Printf("%+v\n", record)
		}
	} else {
		if len(client_name) != 0 {
			var HttpRequestUrl = "http://www.nif.pt/?json=1&q=" + client_name + "&key=" + api_key

			resp, err := http.Get(HttpRequestUrl)
			if err != nil {
				log.Printf("ERROR: Failed to make request %s: %v\n", HttpRequestUrl, err)
				http.Error(w, "failed to make request", http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			var apiResp ApiResponse
			if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
				log.Printf("ERROR: Failed to decode JSON: %v\n", err)
				http.Error(w, "failed to decode response", http.StatusInternalServerError)
				return
			}

			for _, raw := range apiResp.Records {
				// intermediate struct to flatten "contacts" and "geo"
				var tmp struct {
					Nif      int      `json:"nif"`
					Title    string   `json:"title"`
					Address  string   `json:"address"`
					Pc4      string   `json:"pc4"`
					Pc3      string   `json:"pc3"`
					City     string   `json:"city"`
					Activity string   `json:"activity"`
					CaeList  []string `json:"cae"`
					Contacts struct {
						Email   string `json:"email"`
						Phone   string `json:"phone"`
						Website string `json:"website"`
						Fax     string `json:"fax"`
					} `json:"contacts"`
					Geo struct {
						Region string `json:"region"`
						County string `json:"county"`
						Parish string `json:"parish"`
					} `json:"geo"`
					Racius string `json:"racius"`
				}

				if err := json.Unmarshal(raw, &tmp); err != nil {
					panic(err)
				}

				// map into your target struct
				record := NifApiResponse{
					Nif:        tmp.Nif,
					Title:      tmp.Title,
					Address:    tmp.Address,
					Pc4:        tmp.Pc4,
					Pc3:        tmp.Pc3,
					City:       tmp.City,
					Activity:   tmp.Activity,
					CaeList:    tmp.CaeList,
					Email:      tmp.Contacts.Email,
					Phone:      tmp.Contacts.Phone,
					Website:    tmp.Contacts.Website,
					Fax:        tmp.Contacts.Fax,
					Region:     tmp.Geo.Region,
					County:     tmp.Geo.County,
					Parish:     tmp.Geo.Parish,
					RaciusLink: tmp.Racius,
				}
			}
		} else {
			log.Printf("ERROR: Client name field empty: %v\n", err)
			http.Error(w, "Client name field empty", http.StatusBadRequest)
			return
		}
	}

	w.Header().Set("Content-Type", "applicaion/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(record); err != nil {
		log.Printf("ERROR: failed to encode response: %v\n", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
