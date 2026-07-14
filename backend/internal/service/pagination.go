package service

// Paginate executes a list function and a count function, returning the results together.
func Paginate[T any](listFn func() ([]T, error), countFn func() (int64, error), label string) ([]T, int64, error) {
	items, err := listFn()
	if err != nil {
		return nil, 0, Wrap(ErrInternal, "get "+label+" failed", err)
	}
	total, err := countFn()
	if err != nil {
		return nil, 0, Wrap(ErrInternal, "count "+label+" failed", err)
	}
	return items, total, nil
}

// ValidatePagination normalizes and validates pagination parameters, returning the offset and normalized values.
func ValidatePagination(page, limit int) (offset, normPage, normLimit int, err error) {
	if limit > 100 {
		return 0, 0, 0, New(ErrInvalidRequest, "invalid limit")
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	return (page - 1) * limit, page, limit, nil
}

// CalcTotalPages calculates the total number of pages for a given total count and page size.
func CalcTotalPages(total int64, limit int) int {
	if limit <= 0 {
		return 0
	}
	return int((total + int64(limit) - 1) / int64(limit))
}
