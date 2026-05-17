package service

func ValidatePagination(page, limit int) (offset, normPage, normLimit int, err error) {
	if limit > 100 {
		return 0, 0, 0, NewServiceError(ErrInvalidRequest, "invalid limit")
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return (page - 1) * limit, page, limit, nil
}

func CalcTotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
