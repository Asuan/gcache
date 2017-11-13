package gcache

//expired returns true if the item has expired.
func (item Item) expired(now int64) bool {
	if item.Expiration == 0 {
		return false
	}
	return now > item.Expiration
}
