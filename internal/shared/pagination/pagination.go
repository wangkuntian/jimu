package pagination

type Pagination struct {
	Page     int `form:"page" binding:"min=1"`
	PageSize int `form:"page_size" binding:"min=1,max=100"`
}

func (p Pagination) GetOffset() int {
	return (p.Page - 1) * p.PageSize
}

func (p Pagination) GetLimit() int {
	return p.PageSize
}
