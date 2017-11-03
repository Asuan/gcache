package cache

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

type TestStruct struct {
	Num      int
	Children []*TestStruct
}

func TestCache(t *testing.T) {
	tc := New(DefaultExpiration, 0)

	a, found := tc.Get("a")
	if found || a != nil {
		t.Error("Getting A found value that shouldn't exist:", a)
	}

	b, found := tc.Get("b")
	if found || b != nil {
		t.Error("Getting B found value that shouldn't exist:", b)
	}

	c, found := tc.Get("c")
	if found || c != nil {
		t.Error("Getting C found value that shouldn't exist:", c)
	}

	tc.Set("a", []byte{1}, DefaultExpiration)
	tc.Set("b", []byte("b"), DefaultExpiration)
	tc.Set("c", []byte("3.5"), DefaultExpiration)

	x, found := tc.Get("a")
	if !found {
		t.Error("a was not found while getting a2")
	}
	if x == nil {
		t.Error("x for a is nil")
	} else if a2 := x.([]byte)[0]; a2+2 != 3 {
		t.Error("a2 (which should be 1) plus 2 does not equal 3; value:", a2)
	}

	x, found = tc.Get("b")
	if !found {
		t.Error("b was not found while getting b2")
	}
	if x == nil {
		t.Error("x for b is nil")
	} else if b2 := string(x.([]byte)); b2+"B" != "bB" {
		t.Error("b2 (which should be b) plus B does not equal bB; value:", b2)
	}

	x, found = tc.Get("c")
	if !found {
		t.Error("c was not found while getting c2")
	}
	if x == nil {
		t.Error("x for c is nil")
	} else if c2, _ := strconv.ParseFloat(string(x.([]byte)), 64); c2+1.2 != 4.7 {
		t.Error("c2 (which should be 3.5) plus 1.2 does not equal 4.7; value:", c2)
	}
}

func TestCacheTimes(t *testing.T) {
	var found bool

	tc := New(50*time.Millisecond, 1*time.Millisecond)
	tc.Set("a", []byte{1}, DefaultExpiration)
	tc.Set("b", []byte{2}, NoExpiration)
	tc.Set("c", []byte{3}, 20*time.Millisecond)
	tc.Set("d", []byte{4}, 70*time.Millisecond)

	<-time.After(25 * time.Millisecond)
	_, found = tc.Get("c")
	if found {
		t.Error("Found c when it should have been automatically deleted")
	}

	<-time.After(30 * time.Millisecond)
	_, found = tc.Get("a")
	if found {
		t.Error("Found a when it should have been automatically deleted")
	}

	_, found = tc.Get("b")
	if !found {
		t.Error("Did not find b even though it was set to never expire")
	}

	_, found = tc.Get("d")
	if !found {
		t.Error("Did not find d even though it was set to expire later than the default")
	}

	<-time.After(20 * time.Millisecond)
	_, found = tc.Get("d")
	if found {
		t.Error("Found d when it should have been automatically deleted (later than the default)")
	}
}

//TODO init tests
//TODO expiration tests

func TestStatistic(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	stStart := tc.Statistic
	if stStart.AddCount != 0 || stStart.DeleteCount != 0 || stStart.DeleteExpired != 0 ||
		stStart.GetCount != 0 || stStart.ItemsCount != 0 ||
		stStart.ReplaceCount != 0 || stStart.SetCount != 0 {
		t.Error("Statistic for new cach was not 0")
	}

	tc.Add("foo", []byte{1}, DefaultExpiration)
	st1 := tc.Statistic
	if (st1.AddCount - stStart.AddCount) != 1 {
		t.Error("Statistic.AddCount for add new item was not increased")
	}
	if (st1.ItemsCount - stStart.ItemsCount) != 1 {
		t.Errorf("Statistic.ItemsCount for add new item was wrong delta: %d, but expected 1", st1.ItemsCount-stStart.ItemsCount)
	}

	tc.Get("foo")
	st1 = tc.Statistic
	if (st1.GetCount - stStart.GetCount) != 1 {
		t.Error("Statistic.GetCount was not increased after Get()")
	}

	tc.Get("foofoo")
	st2 := tc.Statistic
	if (st2.GetCount - st1.GetCount) != 0 {
		t.Error("Statistic.GetCount was increased after error Get()")
	}
	if (st2.ErrorGetCount - st1.ErrorGetCount) != 1 {
		t.Error("Statistic.ErrorGetCount was not increased after error Get()")
	}

	tc.Replace("foofoo", []byte("bar"), DefaultExpiration)
	if (tc.Statistic.ReplaceCount - st2.ReplaceCount) != 0 {
		t.Error("Statistic.ReplaceCount was increased after error Replace()")
	}
	tc.Replace("foo", []byte("bar"), DefaultExpiration)
	if (tc.Statistic.ReplaceCount - st2.ReplaceCount) != 1 {
		t.Error("Statistic.ReplaceCount was not increased after Replace()")
	}

	tc.Delete("foo1")
	if (tc.Statistic.DeleteCount - st2.DeleteCount) != 0 {
		t.Error("Statistic.DeleteCount was increased after error Delete()")
	}
	if (st2.ItemsCount - tc.Statistic.ItemsCount) != 0 {
		t.Error("Statistic.DeleteCount was changed after error Delete()")
	}
	tc.Delete("foo")
	if (tc.Statistic.DeleteCount - st2.DeleteCount) != 1 {
		t.Error("Statistic.DeleteCount was not increased after Delete()")
	}
	if (st2.ItemsCount - tc.Statistic.ItemsCount) != 1 {
		t.Error("Statistic.DeleteCount was not changed after Delete()")
	}
}

func TestAdd(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	err := tc.Add("foo", []byte("bar"), DefaultExpiration)
	if err != nil {
		t.Error("Couldn't add foo even though it shouldn't exist")
	}
	err = tc.Add("foo", []byte("baz"), DefaultExpiration)
	if err == nil {
		t.Error("Successfully added another foo when it should have returned an error")
	}
}

func TestReplace(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	err := tc.Replace("foo", []byte("bar"), DefaultExpiration)
	if err == nil {
		t.Error("Replaced foo when it shouldn't exist")
	}
	tc.Set("foo", []byte("bar"), DefaultExpiration)
	err = tc.Replace("foo", []byte("bar"), DefaultExpiration)
	if err != nil {
		t.Error("Couldn't replace existing key foo")
	}
}

func TestDelete(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	tc.Set("foo", []byte("bar"), DefaultExpiration)
	tc.Delete("foo")
	x, found := tc.Get("foo")
	if found {
		t.Error("foo was found, but it should have been deleted")
	}
	if x != nil {
		t.Error("x is not nil:", x)
	}
}

func TestItemCount(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	tc.Set("foo", []byte{1}, DefaultExpiration)
	tc.Set("bar", []byte{2}, DefaultExpiration)
	tc.Set("baz", []byte{3}, DefaultExpiration)
	if n := tc.ItemCount(); n != 3 {
		t.Errorf("Item count is not 3: %d", n)
	}
}

func TestFlush(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	tc.Set("foo", []byte("bar"), DefaultExpiration)
	tc.Set("baz", []byte("yes"), DefaultExpiration)
	tc.Flush()
	x, found := tc.Get("foo")
	if found {
		t.Error("foo was found, but it should have been deleted")
	}
	if x != nil {
		t.Error("x is not nil:", x)
	}
	x, found = tc.Get("baz")
	if found {
		t.Error("baz was found, but it should have been deleted")
	}
	if x != nil {
		t.Error("x is not nil:", x)
	}
}

/*
TODO: Save load cache
func TestCacheSerialization(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	testFillAndSerialize(t, tc)

	// Check if gob.Register behaves properly even after multiple gob.Register
	// on c.Items (many of which will be the same type)
	testFillAndSerialize(t, tc)
}

func testFillAndSerialize(t *testing.T, tc *Cache) {
	tc.Set("a", "a", DefaultExpiration)
	tc.Set("b", "b", DefaultExpiration)
	tc.Set("c", "c", DefaultExpiration)
	tc.Set("expired", "foo", 1*time.Millisecond)
	tc.Set("*struct", &TestStruct{Num: 1}, DefaultExpiration)
	tc.Set("[]struct", []TestStruct{
		{Num: 2},
		{Num: 3},
	}, DefaultExpiration)
	tc.Set("[]*struct", []*TestStruct{
		&TestStruct{Num: 4},
		&TestStruct{Num: 5},
	}, DefaultExpiration)
	tc.Set("structception", &TestStruct{
		Num: 42,
		Children: []*TestStruct{
			&TestStruct{Num: 6174},
			&TestStruct{Num: 4716},
		},
	}, DefaultExpiration)

	fp := &bytes.Buffer{}
	err := tc.Save(fp)
	if err != nil {
		t.Fatal("Couldn't save cache to fp:", err)
	}

	oc := New(DefaultExpiration, 0)
	err = oc.Load(fp)
	if err != nil {
		t.Fatal("Couldn't load cache from fp:", err)
	}

	a, found := oc.Get("a")
	if !found {
		t.Error("a was not found")
	}
	if a.(string) != "a" {
		t.Error("a is not a")
	}

	b, found := oc.Get("b")
	if !found {
		t.Error("b was not found")
	}
	if b.(string) != "b" {
		t.Error("b is not b")
	}

	c, found := oc.Get("c")
	if !found {
		t.Error("c was not found")
	}
	if c.(string) != "c" {
		t.Error("c is not c")
	}

	<-time.After(5 * time.Millisecond)
	_, found = oc.Get("expired")
	if found {
		t.Error("expired was found")
	}

	s1, found := oc.Get("*struct")
	if !found {
		t.Error("*struct was not found")
	}
	if s1.(*TestStruct).Num != 1 {
		t.Error("*struct.Num is not 1")
	}

	s2, found := oc.Get("[]struct")
	if !found {
		t.Error("[]struct was not found")
	}
	s2r := s2.([]TestStruct)
	if len(s2r) != 2 {
		t.Error("Length of s2r is not 2")
	}
	if s2r[0].Num != 2 {
		t.Error("s2r[0].Num is not 2")
	}
	if s2r[1].Num != 3 {
		t.Error("s2r[1].Num is not 3")
	}

	s3, found := oc.get("[]*struct")
	if !found {
		t.Error("[]*struct was not found")
	}
	s3r := s3.([]*TestStruct)
	if len(s3r) != 2 {
		t.Error("Length of s3r is not 2")
	}
	if s3r[0].Num != 4 {
		t.Error("s3r[0].Num is not 4")
	}
	if s3r[1].Num != 5 {
		t.Error("s3r[1].Num is not 5")
	}

	s4, found := oc.get("structception")
	if !found {
		t.Error("structception was not found")
	}
	s4r := s4.(*TestStruct)
	if len(s4r.Children) != 2 {
		t.Error("Length of s4r.Children is not 2")
	}
	if s4r.Children[0].Num != 6174 {
		t.Error("s4r.Children[0].Num is not 6174")
	}
	if s4r.Children[1].Num != 4716 {
		t.Error("s4r.Children[1].Num is not 4716")
	}
}


func TestFileSerialization(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	tc.Add("a", "a", DefaultExpiration)
	tc.Add("b", "b", DefaultExpiration)
	f, err := ioutil.TempFile("", "go-cache-cache.dat")
	if err != nil {
		t.Fatal("Couldn't create cache file:", err)
	}
	fname := f.Name()
	f.Close()
	tc.SaveFile(fname)

	oc := New(DefaultExpiration, 0)
	oc.Add("a", "aa", 0) // this should not be overwritten
	err = oc.LoadFile(fname)
	if err != nil {
		t.Error(err)
	}
	a, found := oc.Get("a")
	if !found {
		t.Error("a was not found")
	}
	astr := a.(string)
	if astr != "aa" {
		if astr == "a" {
			t.Error("a was overwritten")
		} else {
			t.Error("a is not aa")
		}
	}
	b, found := oc.Get("b")
	if !found {
		t.Error("b was not found")
	}
	if b.(string) != "b" {
		t.Error("b is not b")
	}
}

func TestSerializeUnserializable(t *testing.T) {
	tc := New(DefaultExpiration, 0)
	ch := make(chan bool, 1)
	ch <- true
	tc.Set("chan", ch, DefaultExpiration)
	fp := &bytes.Buffer{}
	err := tc.Save(fp) // this should fail gracefully
	if err.Error() != "gob NewTypeObject can't handle type: chan bool" {
		t.Error("Error from Save was not gob NewTypeObject can't handle type chan bool:", err)
	}
}
*/
func BenchmarkCacheGetExpiring(b *testing.B) {
	benchmarkCacheGet(b, 5*time.Minute)
}

func BenchmarkCacheGetNotExpiring(b *testing.B) {
	benchmarkCacheGet(b, NoExpiration)
}

func benchmarkCacheGet(b *testing.B, exp time.Duration) {
	b.StopTimer()
	tc := New(exp, 0)
	tc.Set("foo", []byte("bar"), DefaultExpiration)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Get("foo")
	}
}

func BenchmarkRWMutexMapGet(b *testing.B) {
	b.StopTimer()
	m := map[string]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m["foo"]
		mu.RUnlock()
	}
}

func BenchmarkRWMutexInterfaceMapGetStruct(b *testing.B) {
	b.StopTimer()
	s := struct{ name string }{name: "foo"}
	m := map[interface{}]string{
		s: "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m[s]
		mu.RUnlock()
	}
}

func BenchmarkRWMutexInterfaceMapGetString(b *testing.B) {
	b.StopTimer()
	m := map[interface{}]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.RLock()
		_, _ = m["foo"]
		mu.RUnlock()
	}
}

func BenchmarkCacheGetConcurrentExpiring(b *testing.B) {
	benchmarkCacheGetConcurrent(b, 5*time.Minute)
}

func BenchmarkCacheGetConcurrentNotExpiring(b *testing.B) {
	benchmarkCacheGetConcurrent(b, NoExpiration)
}

func benchmarkCacheGetConcurrent(b *testing.B, exp time.Duration) {
	b.StopTimer()
	tc := New(exp, 0)
	tc.Set("foo", []byte("bar"), DefaultExpiration)
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get("foo")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkRWMutexMapGetConcurrent(b *testing.B) {
	b.StopTimer()
	m := map[string]string{
		"foo": "bar",
	}
	mu := sync.RWMutex{}
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				mu.RLock()
				_, _ = m["foo"]
				mu.RUnlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkCacheGetManyConcurrentExpiring(b *testing.B) {
	benchmarkCacheGetManyConcurrent(b, 5*time.Minute)
}

func BenchmarkCacheGetManyConcurrentNotExpiring(b *testing.B) {
	benchmarkCacheGetManyConcurrent(b, NoExpiration)
}

func benchmarkCacheGetManyConcurrent(b *testing.B, exp time.Duration) {
	// This is the same as BenchmarkCacheGetConcurrent, but its result
	// can be compared against BenchmarkShardedCacheGetManyConcurrent
	// in sharded_test.go.
	b.StopTimer()
	n := 10000
	tc := New(exp, 0)
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		k := "foo" + strconv.Itoa(n)
		keys[i] = k
		tc.Set(k, []byte("bar"), DefaultExpiration)
	}
	each := b.N / n
	wg := new(sync.WaitGroup)
	wg.Add(n)
	for _, v := range keys {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get(v)
			}
			wg.Done()
		}()
	}
	b.StartTimer()
	wg.Wait()
}

func BenchmarkCacheSetExpiring(b *testing.B) {
	benchmarkCacheSet(b, 5*time.Minute)
}

func BenchmarkCacheSetNotExpiring(b *testing.B) {
	benchmarkCacheSet(b, NoExpiration)
}

func benchmarkCacheSet(b *testing.B, exp time.Duration) {
	b.StopTimer()
	tc := New(exp, 0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", []byte("bar"), DefaultExpiration)
	}
}

func BenchmarkRWMutexMapSet(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
	}
}

func BenchmarkCacheSetDelete(b *testing.B) {
	b.StopTimer()
	tc := New(DefaultExpiration, 0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", []byte("bar"), DefaultExpiration)
		tc.Delete("foo")
	}
}

func BenchmarkRWMutexMapSetDelete(b *testing.B) {
	b.StopTimer()
	m := map[string]string{}
	mu := sync.RWMutex{}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		mu.Lock()
		m["foo"] = "bar"
		mu.Unlock()
		mu.Lock()
		delete(m, "foo")
		mu.Unlock()
	}
}

func BenchmarkDeleteExpiredLoop(b *testing.B) {
	b.StopTimer()
	tc := New(5*time.Minute, 0)

	for i := 0; i < 100000; i++ {
		tc.Set(strconv.Itoa(i), []byte("bar"), DefaultExpiration)
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.DeleteExpired()
	}
}
