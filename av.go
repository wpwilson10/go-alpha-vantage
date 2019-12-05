package av

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

const (
	// HostDefault is the default host for Alpha Vantage
	HostDefault = "www.alphavantage.co"
)

const (
	schemeHttps = "https"

	queryApiKey     = "apikey"
	queryDataType   = "datatype"
	queryOutputSize = "outputsize"
	querySymbol     = "symbol"
	queryMarket     = "market"
	queryEndpoint   = "function"
	queryInterval   = "interval"
	queryKeywords   = "keywords"

	valueCompact                  = "compact"
	valueFull                     = "full"
	valueJson                     = "json"
	valueDigitcalCurrencyEndpoint = "DIGITAL_CURRENCY_INTRADAY"
	valueSymbolSearchEndpoint     = "SYMBOL_SEARCH"

	pathQuery = "query"

	requestTimeout = time.Second * 30
)

type OutputSize uint8

const (
	Compact OutputSize = iota
	Full
)

func (outputSize OutputSize) String() string {
	switch outputSize {
	case Compact:
		return valueCompact
	case Full:
		return valueFull
	}
	// default to compact if a non expected value is passed
	return valueCompact
}

// Connection is an interface that requests data from a server
type Connection interface {
	// Request creates an http Response from the given endpoint URL
	Request(endpoint *url.URL) (*http.Response, error)
}

type avConnection struct {
	client *http.Client
	host   string
}

// NewConnectionHost creates a new connection at the default Alpha Vantage host
func NewConnection() Connection {
	return NewConnectionHost(HostDefault)
}

// NewConnectionHost creates a new connection at the given host
func NewConnectionHost(host string) Connection {
	client := &http.Client{
		Timeout: requestTimeout,
	}
	return &avConnection{
		client: client,
		host:   host,
	}
}

// Request will make an HTTP GET request for the given endpoint from Alpha Vantage
func (conn *avConnection) Request(endpoint *url.URL) (*http.Response, error) {
	endpoint.Scheme = schemeHttps
	endpoint.Host = conn.host
	targetUrl := endpoint.String()
	return conn.client.Get(targetUrl)
}

// Client is a service used to query Alpha Vantage stock data
type Client struct {
	conn   Connection
	apiKey string
}

// NewClientConnection creates a new Client with the default Alpha Vantage connection
func NewClient(apiKey string) *Client {
	return NewClientConnection(apiKey, NewConnection())
}

// NewClientConnection creates a Client with a specific connection
func NewClientConnection(apiKey string, connection Connection) *Client {
	return &Client{
		conn:   connection,
		apiKey: apiKey,
	}
}

// buildRequestPath builds an endpoint URL with the given query parameters
func (c *Client) buildRequestPath(params map[string]string) *url.URL {
	// build our URL
	endpoint := &url.URL{}
	endpoint.Path = pathQuery

	// base parameters
	query := endpoint.Query()
	query.Set(queryApiKey, c.apiKey)
	query.Set(queryDataType, valueCsv)
	query.Set(queryOutputSize, valueCompact)

	// additional parameters
	for key, value := range params {
		query.Set(key, value)
	}

	endpoint.RawQuery = query.Encode()

	return endpoint
}

// StockTimeSeriesIntraday queries a stock symbols statistics throughout the day.
// Data is returned from past to present.
func (c *Client) StockTimeSeriesIntraday(timeInterval TimeInterval, symbol string) ([]*TimeSeriesValue, error) {
	endpoint := c.buildRequestPath(map[string]string{
		queryEndpoint: timeSeriesIntraday.keyName(),
		queryInterval: timeInterval.keyName(),
		querySymbol:   symbol,
	})
	response, err := c.conn.Request(endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return parseTimeSeriesData(response.Body)
}

// StockTimeSeries queries a stock symbols statistics for a given time frame.
// Optionally an OutputSize can be specified. Only the first optional outputSize will be used.
// Data is returned from past to present.
func (c *Client) StockTimeSeries(timeSeries TimeSeries, symbol string, optionalOutputSize ...OutputSize) ([]*TimeSeriesValue, error) {
	endpoint := c.buildRequestPath(map[string]string{
		queryEndpoint: timeSeries.keyName(),
		querySymbol:   symbol,
		queryOutputSize: getOutputSize(optionalOutputSize),
	})
	response, err := c.conn.Request(endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return parseTimeSeriesData(response.Body)
}

func getOutputSize(optionalOutputSize []OutputSize) string {
	if len(optionalOutputSize) > 0 {
		return optionalOutputSize[0].String()
	}

	return Compact.String()
}

// DigitalCurrency queries statistics of a digital currency in terms of a physical currency throughout the day.
// Data is returned from past to present.
func (c *Client) DigitalCurrency(digital, physical string) ([]*DigitalCurrencySeriesValue, error) {
	endpoint := c.buildRequestPath(map[string]string{
		queryEndpoint: valueDigitcalCurrencyEndpoint,
		querySymbol:   digital,
		queryMarket:   physical,
	})
	response, err := c.conn.Request(endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return parseDigitalCurrencySeriesData(response.Body)
}

func (c *Client) SymbolSearch(keywords string) (*SymbolMatches, error) {
	endpoint := c.buildRequestPath(map[string]string{
		queryEndpoint: valueSymbolSearchEndpoint,
		queryDataType: valueJson,
		queryKeywords: keywords,
	})

	response, err := c.conn.Request(endpoint)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	var matches *SymbolMatches
	json.Unmarshal(body, &matches)

	return matches, nil
}
	
// StockQuote is a lightweight alternative to the time series APIs, this service returns the latest price and volume
// information for a security of your choice.
func (c *Client) StockQuote(symbol string) (*QuoteValue, error) {
	endpoint := c.buildRequestPath(map[string]string{
		queryEndpoint: GlobalQuote,
		querySymbol:   symbol,
	})
	response, err := c.conn.Request(endpoint)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return parseQuoteData(response.Body)
}
