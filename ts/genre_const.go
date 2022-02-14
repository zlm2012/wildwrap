package ts

import "strconv"

const (
	News Genre = iota << 4
	Sports
	Information
	Drama
	Music
	Variety
	Movies
	AnimationSEMovies
	DocumentaryCulture
	TheaterPublicPerformance
	HobbyEducation
	Welfare
	GenreReserved1
	GenreReserved2
	GenreExtension
	GenreOthers
)

const (
	NewsRegular SubGenre = iota
	NewsWeather
	NewsDocumentary
	NewsPolitics
	NewsEconomics
	NewsInternational
	NewsAnalysis
	NewsDiscussion
	NewsSpecial
	NewsLocal
	NewsTraffic
)

const (
	SportsNews SubGenre = iota + 0x10
	SportsBaseball
	SportsSoccer
	SportsGolf
	SportsOtherBallGames
	SportsSumoCombative
	SportsOlympicsInternationalGames
	SportsAthleticSwimming
	SportsMotor
	SportsMarineWinter
	SportsRace
)

const (
	AnimeJapanese SubGenre = iota + 0x70
	AnimeOverseas
	SpecialEffects
)

func (v SubGenre) String() string {
	return "0x" + strconv.FormatUint(uint64(v), 16)
}
