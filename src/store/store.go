package store

import "errors"

type Store struct {
	Store      string
	LocationID int
	URL        string
	Staffs     map[int]*Staff // key: StaffMemberID
}

type Staff struct {
	Staff         string
	StaffMemberID int
	Treatments    map[int]*Treatment // key: TreatmentID
}

type Treatment struct {
	Treatment   string
	TreatmentID int
}

var Stores map[string]*Store

func init() {
	Stores = make(map[string]*Store)
}

func AddNewStore(store string, id int, url string) error {
	Stores[store] = &Store{Store: store, LocationID: id, URL: url}
	return nil
}

func (s *Store) AddStaff(store string, id int, url string) error {
	if s == nil {
		return errors.New("nil")
	}
	s.Staffs[id] = &Staff{}
	return nil
}
