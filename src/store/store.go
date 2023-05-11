package store

import (
	"errors"
	"sync"
)

type Store struct {
	URL          string
	LocationID   int
	Link         string //office website
	CalendarLink string
	Staffs       map[string]*Staff   // key: StaffMemberID
	Disciplines  map[int]*Discipline //key: discipline id
}
type Discipline struct {
	Name       string
	ID         int
	Treatments map[int]*Treatment // key: treament id
}

type Staff struct {
	Staff         string
	StaffMemberID string
	Treatments    map[int]*Treatment // key: TreatmentID
}

type Treatment struct {
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

func AddStore(url string, id int, link string) error {
	if stores[url] == nil {
		stores[url] = &Store{URL: url, LocationID: id, Link: link, Staffs: make(map[string]*Staff), Disciplines: make(map[int]*Discipline)}
	} else {
		stores[url].URL = url
		stores[url].LocationID = id
		stores[url].URL = url
	}
	return nil
}

func (s *Store) AddDisciplines(id int, name string) error {
	mutex.Lock()
	defer mutex.Unlock()
	if s == nil {
		return errors.New("nil")
	}
	s.Disciplines[id] = &Discipline{ID: id, Name: name}
	return nil
}

func (d *Discipline) AddTreatment(id, duration int, fee float64) error {
	mutex.Lock()
	defer mutex.Unlock()
	if d == nil {
		return errors.New("nil")
	}
	if d.Treatments == nil {
		d.Treatments = make(map[int]*Treatment)
	}

	d.Treatments[id] = &Treatment{ID: id, Duration: duration, Fee: fee}
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
