package models

import "history-api/internal/dtos/response"

type UserProfileSimple struct {
	DisplayName string `json:"display_name"`
	FullName    string `json:"full_name"`
	AvatarUrl   string `json:"avatar_url"`
	Bio         string `json:"bio"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	CountryCode string `json:"country_code"`
	Phone       string `json:"phone"`
}

func (p *UserProfileSimple) ToResponse() *response.UserProfileSimpleResponse {
	if p == nil {
		return nil
	}
	return &response.UserProfileSimpleResponse{
		DisplayName: p.DisplayName,
		FullName:    p.FullName,
		AvatarUrl:   p.AvatarUrl,
		Bio:         p.Bio,
		Location:    p.Location,
		Website:     p.Website,
		CountryCode: p.CountryCode,
		Phone:       p.Phone,
	}
}
