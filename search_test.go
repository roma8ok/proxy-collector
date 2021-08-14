package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestMakeDDGSearchURL(t *testing.T) {
	rightQueryURL := "https://html.duckduckgo.com/html?q=proxy+list"

	query := makeDDGSearchURL("proxy list")
	if query != rightQueryURL {
		t.Errorf(`makeDDGSearchURL("proxy list") = %s, expected %s`, query, rightQueryURL)
	}
}

func TestFindSiteURLsFromDDG_EmptyHTML(t *testing.T) {
	expected := make([]string, 0)

	got := findSiteURLsFromDDG([]byte{})
	if len(got) != 0 {
		t.Errorf(`findSiteURLsFromDDG("") = %v, expected %v`, got, expected)
	}
}

func TestFindSiteURLsFromDDG_Success(t *testing.T) {
	expected := []string{
		"https://www.merriam-webster.com/dictionary/query",
		"https://support.microsoft.com/en-us/office/introduction-to-queries-a9739a09-d3ff-4f36-8ac3-5760249fb65c",
		"https://www.dictionary.com/browse/query",
		"https://www.thefreedictionary.com/query",
		"https://www.webopedia.com/definitions/query/",
		"https://dictionary.cambridge.org/dictionary/english/query",
		"https://www.techopedia.com/definition/5736/query",
		"https://support.microsoft.com/en-us/office/create-a-simple-select-query-de8b1c8d-14e9-4b25-8e22-70888d54de59",
		"https://docs.microsoft.com/en-us/sql/t-sql/xml/query-method-xml-data-type",
		"https://www.baeldung.com/spring-data-jpa-query",
		"https://docs.microsoft.com/en-us/azure/cosmos-db/sql-query-getting-started",
		"https://www.merriam-webster.com/thesaurus/query",
		"https://www.imdb.com/title/tt11086128/",
		"https://www.educative.io/blog/what-is-database-query-sql-nosql",
		"https://support.google.com/docs/answer/3093343?hl=en",
		"https://www.thefreedictionary.com/queries",
		"https://www.vocabulary.com/dictionary/query",
		"https://docs.aws.amazon.com/amazondynamodb/latest/APIReference/API_Query.html",
		"https://legal-dictionary.thefreedictionary.com/query",
		"https://dictionary.cambridge.org/vi/dictionary/english/query",
		"https://firebase.google.com/docs/reference/android/com/google/firebase/database/Query",
		"https://clearinghouse.fmcsa.dot.gov/Query/Plan",
		"https://developer.android.com/reference/android/arch/persistence/room/Query",
		"https://medical-dictionary.thefreedictionary.com/query",
		"https://www.collinsdictionary.com/dictionary/english/query",
		"https://www.thesaurus.com/browse/query",
		"https://financial-dictionary.thefreedictionary.com/Query",
		"https://cloud.google.com/bigquery/docs/reference/standard-sql/query-syntax",
		"https://en.wikipedia.org/wiki/Query_language",
		"https://www.petrikainulainen.net/programming/spring-framework/spring-data-jpa-tutorial-creating-database-queries-with-the-query-annotation/",
	}

	got := findSiteURLsFromDDG([]byte(duckDuckGoSearchHTML))
	if !reflect.DeepEqual(expected, got) {
		t.Errorf(`findSiteURLsFromDDG(html) = %v, expected %v`, got, expected)
	}
}

func TestFindURLs_Empty(t *testing.T) {
	empty := make([]byte, 0)
	urls := findURLs(empty)
	if len(urls) != 0 {
		t.Errorf(`findURLs(%s) = %v, expected empty slice`, empty, urls)
	}
}

func TestFindURLs_Success(t *testing.T) {
	in := `
Found:
1. http://www.youtube.com
2. https://www.facebook.com
3. http://baidu.com
4. https://yahoo.com
5. http://www.amazon.com?query=123
6. http://api.wikipedia.org/
7. https://www.google.co.in

Not found, because they don't have a protocol:
1. qq.com
2. twitter.com/123/
3. live.com?query=123
`

	expected := []string{
		"http://www.youtube.com",
		"https://www.facebook.com",
		"http://baidu.com",
		"https://yahoo.com",
		"http://www.amazon.com?query=123",
		"http://api.wikipedia.org/",
		"https://www.google.co.in",
	}

	got := findURLs([]byte(in))
	if !reflect.DeepEqual(got, expected) {
		t.Errorf(`findURLs(data) = %v, expected %v`, got, expected)
	}
}

func TestSendGetRequest_Success(t *testing.T) {
	const successBody = "OK"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, successBody)
	}))
	defer server.Close()

	logger, _ := newLogger("", "")
	headers := make(map[string]string)

	body, err := sendGetRequest(logger, server.URL, "", headers)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, headers, err)
	}
	if string(body) != successBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, headers, string(body), successBody)
	}
}

func TestSendGetRequest_Headers(t *testing.T) {
	const (
		successBody = "OK"
		failureBody = "wrong_headers"
		headerName  = "rightHeader"
		headerValue = "right_value"
	)
	rightHeaders := map[string]string{
		headerName: headerValue,
	}
	wrongHeaders := map[string]string{
		"wrongHeader": "wrong_value",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerName) == headerValue {
			_, _ = io.WriteString(w, successBody)
		} else {
			_, _ = io.WriteString(w, failureBody)
		}
	}))
	defer server.Close()

	logger, _ := newLogger("", "")

	body, err := sendGetRequest(logger, server.URL, "", rightHeaders)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, rightHeaders, err)
	}
	if string(body) != successBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, rightHeaders, string(body), successBody)
	}

	body, err = sendGetRequest(logger, server.URL, "", wrongHeaders)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, wrongHeaders, err)
	}
	if string(body) != failureBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, wrongHeaders, string(body), failureBody)
	}
}

func TestSendGetRequest_StatusCode(t *testing.T) {
	const (
		successBody = "OK"
		failureBody = "wrong_headers"
		headerName  = "rightHeader"
		headerValue = "right_value"
	)
	rightHeaders := map[string]string{
		headerName: headerValue,
	}
	wrongHeaders := map[string]string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(headerName) == headerValue {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, successBody)
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, failureBody)
		}
	}))
	defer server.Close()

	logger, _ := newLogger("", "")

	body, err := sendGetRequest(logger, server.URL, "", rightHeaders)
	if err != nil {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) error: %s`, server.URL, rightHeaders, err)
	}
	if string(body) != successBody {
		t.Errorf(`sendGetRequest(logger, %s, "", %s) = "%s", expected "%s"`, server.URL, rightHeaders, string(body), successBody)
	}

	body, err = sendGetRequest(logger, server.URL, "", wrongHeaders)
	if err != errStatusCodeNotOK {
		t.Errorf(`sendGetRequest(logger, %s, "", %s); expected error: %s, got: %s`, server.URL, wrongHeaders, errStatusCodeNotOK, err)
	}
}
