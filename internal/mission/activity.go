package mission

import "time"

type CrewActivity struct {
	StartMET time.Duration
	EndMET   time.Duration
	Label    string
	Detail   string
}

// CrewActivities is an approximate crew-facing schedule derived from NASA's
// Artemis II overview timeline PDF. It is intentionally coarser than the
// milestone table and is used for "what's happening now / what's next" views.
var CrewActivities = []CrewActivity{
	{d(1, 1, 0), d(1, 1, 45), "TLI", "Crew operations during translunar injection"},
	{d(1, 1, 45), d(1, 3, 0), "Meal", "Flight day meal block"},
	{d(1, 3, 0), d(1, 3, 15), "Pulse Ox", "Crew pulse oximetry check"},
	{d(1, 3, 15), d(1, 3, 30), "PWD", "Private medical conference / crew ops"},
	{d(1, 3, 30), d(1, 4, 0), "PAO", "Public affairs event"},
	{d(1, 5, 0), d(1, 6, 0), "Wnd Inspect", "Window inspection"},
	{d(1, 9, 0), d(1, 17, 30), "Sleep", "Crew sleep block"},

	{d(2, 0, 0), d(2, 1, 0), "OTC-1", "Outbound trajectory correction operations"},
	{d(2, 1, 0), d(2, 2, 0), "Meal", "Flight day meal block"},
	{d(2, 2, 0), d(2, 2, 30), "PAO", "Public affairs event"},
	{d(2, 2, 30), d(2, 3, 15), "CPR Demo", "Crew procedure demo"},
	{d(2, 3, 15), d(2, 4, 0), "Med Kit", "Medical kit operations"},
	{d(2, 4, 0), d(2, 4, 30), "NatGeo", "National Geographic segment"},
	{d(2, 4, 30), d(2, 5, 0), "Lunar Cab", "Cabin / lunar outreach activity"},
	{d(2, 5, 0), d(2, 6, 0), "DSN Emer Comm", "Emergency comms demo"},
	{d(2, 6, 0), d(2, 6, 30), "D5 Cam Window", "Camera window activity"},
	{d(2, 9, 0), d(2, 17, 30), "Sleep", "Crew sleep block"},

	{d(3, 0, 0), d(3, 1, 0), "OTC-2", "Outbound trajectory correction operations"},
	{d(3, 1, 0), d(3, 1, 45), "COGN", "Cognitive task block"},
	{d(3, 1, 45), d(3, 2, 45), "Meal", "Flight day meal block"},
	{d(3, 2, 45), d(3, 3, 15), "PAO", "Public affairs event"},
	{d(3, 3, 15), d(3, 4, 0), "Doc Cam Measure", "Docking camera measurement activity"},
	{d(3, 4, 30), d(3, 5, 45), "Lunar Img Review", "Lunar imagery review"},
	{d(3, 5, 45), d(3, 6, 15), "PWD", "Private crew conference"},
	{d(3, 6, 15), d(3, 6, 45), "Lunar Img", "Lunar imagery activity"},
	{d(3, 9, 0), d(3, 17, 30), "Sleep", "Crew sleep block"},

	{d(4, 0, 0), d(4, 3, 0), "OCSS DFTO Ops", "Crew survival systems operations"},
	{d(4, 3, 0), d(4, 3, 30), "Depress Ops", "Cabin depressurization operations"},
	{d(4, 3, 30), d(4, 4, 15), "Meal", "Flight day meal block"},
	{d(4, 4, 15), d(4, 4, 45), "OTC-3", "Outbound trajectory correction operations"},
	{d(4, 4, 45), d(4, 5, 15), "SAT Mode Test", "Communications mode test"},
	{d(4, 9, 0), d(4, 17, 30), "Sleep", "Crew sleep block"},

	{d(5, 0, 0), d(5, 2, 30), "Lunar Obs", "Lunar observation block"},
	{d(5, 2, 30), d(5, 3, 30), "Meal", "Flight day meal block"},
	{d(5, 4, 0), d(5, 5, 0), "Lunar Doc Brief", "Lunar documentation / brief"},
	{d(5, 5, 0), d(5, 5, 30), "PAO", "Public affairs event"},
	{d(5, 5, 30), d(5, 6, 45), "Exer Noise", "Exercise / acoustic operations"},
	{d(5, 9, 0), d(5, 17, 30), "Sleep", "Crew sleep block"},

	{d(6, 0, 0), d(6, 1, 0), "Off Duty", "Crew off-duty period"},
	{d(6, 1, 0), d(6, 2, 15), "PTV Exercise", "Physical training / exercise"},
	{d(6, 2, 15), d(6, 3, 0), "Meal", "Flight day meal block"},
	{d(6, 3, 0), d(6, 4, 0), "Off Duty", "Crew off-duty period"},
	{d(6, 4, 0), d(6, 4, 30), "RTC-1", "Return trajectory correction operations"},
	{d(6, 4, 30), d(6, 6, 0), "Off Duty", "Crew off-duty period"},
	{d(6, 9, 0), d(6, 18, 0), "Sleep", "Crew sleep / shift handover"},

	{d(7, 0, 0), d(7, 0, 45), "Press", "Media / outreach block"},
	{d(7, 0, 45), d(7, 1, 30), "NG", "National Geographic segment"},
	{d(7, 1, 30), d(7, 2, 15), "Meal", "Flight day meal block"},
	{d(7, 2, 15), d(7, 4, 0), "Rad Shelter Demo", "Radiation shelter demonstration"},
	{d(7, 4, 0), d(7, 5, 15), "Man PLT DFTO", "Manual piloting demonstration"},
	{d(7, 9, 0), d(7, 17, 30), "Sleep", "Crew sleep block"},

	{d(8, 0, 0), d(8, 0, 45), "PAO", "Public affairs event"},
	{d(8, 0, 45), d(8, 1, 30), "Meal", "Flight day meal block"},
	{d(8, 1, 30), d(8, 3, 0), "OIG Don DFTO", "Entry suit / checkouts"},
	{d(8, 3, 0), d(8, 3, 30), "RTC-2", "Return trajectory correction operations"},
	{d(8, 9, 0), d(8, 17, 30), "Sleep", "Crew sleep block"},

	{d(9, 0, 0), d(9, 0, 45), "RTC-3", "Return trajectory correction operations"},
	{d(9, 0, 45), d(9, 2, 0), "Cabin Config", "Cabin configuration for entry"},
	{d(9, 2, 0), d(9, 3, 0), "Entry C/L", "Entry checklist"},
	{d(9, 3, 0), d(9, 9, 0), "Recovery Ops", "Recovery operations after splashdown"},
}

func CurrentCrewActivity(met time.Duration) *CrewActivity {
	for i := range CrewActivities {
		a := &CrewActivities[i]
		if met >= a.StartMET && met < a.EndMET {
			return a
		}
	}
	return nil
}

func NextCrewActivity(met time.Duration) *CrewActivity {
	for i := range CrewActivities {
		a := &CrewActivities[i]
		if met < a.StartMET {
			return a
		}
	}
	return nil
}
