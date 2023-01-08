package fuzzer

type Filters struct {
	StatusCodes []int `json:"statusCodes"`
	Words       []int `json:"words"`
	Lines       []int `json:"lines"`
	Size        []int `json:"size"`
}

func (f *Fuzzer) filterResult(lines, words, size, statusCode int) (res bool) {

	for _, c := range f.Filters.StatusCodes {
		if c == statusCode {
			return false
		}
	}

	for _, l := range f.Filters.Lines {
		if l == lines {
			return false
		}
	}

	for _, w := range f.Filters.Words {
		if w == words {
			return false
		}
	}

	for _, s := range f.Filters.Size {
		if s == size {
			return false
		}
	}

	return true
}
