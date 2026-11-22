package model

// FuxiHallContributions caches history; CorpPAP is read-only and has no manual offset.
type FuxiHallContributions struct {
	CorpPAP     float64 `json:"corp_pap"`
	Fleets      int64   `json:"fleets"`
	Welfare     int64   `json:"welfare"`
	SRP         int64   `json:"srp"`
	Recruitment int64   `json:"recruitment"`
	Mentoring   int64   `json:"mentoring"`
}

func (c FuxiHallCard) Offsets() FuxiHallContributions {
	return FuxiHallContributions{Fleets: c.FleetLedOffset, Welfare: int64(c.WelfareDeliveryOffset), SRP: c.SRPOffset, Recruitment: c.RecruitmentOffset, Mentoring: c.MentoringOffset}
}

func (c *FuxiHallCard) EnrichContributions() {
	o := c.Offsets()
	raw := c.ContributionCounts
	c.Contributions = FuxiHallContributions{Fleets: raw.Fleets + o.Fleets, Welfare: raw.Welfare + o.Welfare, SRP: raw.SRP + o.SRP, Recruitment: raw.Recruitment + o.Recruitment, Mentoring: raw.Mentoring + o.Mentoring}
	c.Contributions.CorpPAP = raw.CorpPAP
}
