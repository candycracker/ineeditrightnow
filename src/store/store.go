package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var merchants = make(map[string]*Merchant)

type Merchant struct {
	Name    string
	Website string
	Stores  []*Store
}

func (m *Merchant) String() string {
	return fmt.Sprintf("Name: %s, Webite: %s, Store: %v", m.Name, m.Website, m.Stores)
}

func (s *Store) Sting() string {
	return fmt.Sprintf("Name: %s, Url: %s, Email: %s, LocationID: %d, Address: %s, Phone: %s, Staffs: %v, Disciplines: %v",
		s.Name, s.Url, s.Email, s.LocationID, s.Address, s.Phone, s.Staffs, s.Disciplines)
}

func convertUrl(url string, locID, discID, trmtID int) string {
	newUrl := fmt.Sprintf("%sapi/v2/openings/for_discipline?location_id=%d&discipline_id=%d&treatment_id=%d&date=&num_days=7", url, locID, discID, trmtID)
	return newUrl
}

func (m *Merchant) Search(treatment string, date time.Time) string {
	// urls := []string{}
	for _, store := range m.Stores {
		for t, disc := range store.Disciplines {
			if strings.Contains(t, treatment) {
				for i, disc := range disc.Treatments {
					url := convertUrl(m.Website, store.LocationID, disc.ID, i)
					// urls = append(urls, url)
					cldns := getCalendar(url)
					for _, cldn := range cldns {
						cldn.Staff = store.Staffs[fmt.Sprint(cldn.StaffMemberID)].Staff
						cldn.Location = store.Address
						cldn.Treatment = disc.Content
						t, err := time.Parse("2006-01-02T15:04:05", cldn.StartAt)
						if err != nil {
							fmt.Println("???", err)
							continue
						}
						if t.Day() == date.Day() {
							fmt.Println(store.Url)
							fmt.Println(cldn.Location)
							fmt.Println(cldn.Treatment, cldn.Staff, cldn.StartAt, cldn.EndAt)
						}

					}

				}
			}
		}
	}

	return ""
}

type Calendar struct {
	StaffMemberID       int `json:"staff_member_id"`
	Staff               string
	LocationID          int `json:"location_id"`
	Location            string
	TreatmentID         int `json:"treatment_id"`
	Treatment           string
	Duration            int    `json:"duration"`
	StartAt             string `json:"start_at"`
	EndAt               string `json:"end_at"`
	RoomID              int    `json:"room_id"`
	CallToBook          bool   `json:"call_to_book"`
	State               string `json:"state"`
	Status              string `json:"status"`
	ParentAppointmentID string `json:"parent_appointment_id"`
}

func getCalendar(url string) []*Calendar {

	data := []*Calendar{}
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return data
	}
	req.Header.Add("Cookie", "_front_desk_session=")
	resp, err := client.Do(req)
	if err != nil {
		return data
	}

	defer resp.Body.Close()

	if resp.StatusCode > 400 {
		fmt.Println(url, "Status code: ", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return data
	}

	json.Unmarshal(body, &data)

	return data
}

func AddNewMerchant(name string, m *Merchant) {
	merchants[name] = m
}

func GetAllMerchants() map[string]*Merchant {
	return merchants
}

func FindMerchant(name string) *Merchant {
	return merchants[name]
}

func MakeNewMerchant(name, website string) error {
	if merchants[name] != nil {
		return errors.New(name + " already existed")
	}
	merchants[name] = &Merchant{name, website, []*Store{}}
	return nil
}

type Store struct {
	Name        string
	Url         string
	Email       string
	LocationID  int
	Address     string
	Phone       string
	StartTime   string
	EndTime     string
	Staffs      map[string]*Staff      // key: StaffMemberID
	Disciplines map[string]*Discipline //key: discipline id
}
type Discipline struct {
	Content    string
	ID         int
	Treatments map[int]*Treatment // key: treament id
}

type Staff struct {
	Staff         string
	StaffMemberID string
	Treatments    []int // key: TreatmentID
}

type Treatment struct {
	Content  string
	ID       int
	Duration int
	Fee      float64
}

var stores map[string]*Store
var mutex *sync.Mutex

func init() {
	stores = make(map[string]*Store)
	mutex = &sync.Mutex{}
}

func GetStore() map[string]*Store {
	return stores
}

func FindStore(jUrl string) *Store {
	return stores[jUrl]
}

func NewStore(name, url string) *Store {
	return &Store{Name: name, Url: url, Staffs: make(map[string]*Staff), Disciplines: make(map[string]*Discipline)}
}

func AddStore(url string, store *Store) error {

	stores[url] = store

	return nil
}

func (s *Store) GetAllTreatments() []string {
	treatments := []string{}
	for t := range s.Disciplines {
		treatments = append(treatments, t)
	}
	return treatments
}

func (s *Store) GetDiscipline(attr string) *Discipline {
	mutex.Lock()
	defer mutex.Unlock()
	return s.Disciplines[attr]
}

func (s *Store) AddNewDiscipline(attr string, id int, content string) *Discipline {
	mutex.Lock()
	defer mutex.Unlock()
	s.Disciplines[attr] = &Discipline{ID: id, Content: content, Treatments: make(map[int]*Treatment)}
	return s.Disciplines[attr]
}

func (d *Discipline) AddNewTreatment(id, duration int, fee float64, content string) error {
	mutex.Lock()
	defer mutex.Unlock()
	d.Treatments[id] = &Treatment{ID: id, Duration: duration, Fee: fee, Content: content}
	return nil
}

func (s *Store) AddStaff(staff string, id string) error {

	mutex.Lock()
	defer mutex.Unlock()
	if s == nil {
		return errors.New("nil")
	}
	s.Staffs[id] = &Staff{Staff: staff, StaffMemberID: id}
	return nil
}

func (s *Staff) AddNewTreatment(id int) {
	s.Treatments = append(s.Treatments, id)
}
