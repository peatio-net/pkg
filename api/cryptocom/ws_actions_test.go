package cryptocom

import (
  "bytes"
  "database/sql"
  "encoding/json"
  "fmt"
  "testing"

  "github.com/bmizerany/assert"
  "github.com/google/uuid"
  "github.com/shopspring/decimal"
)

type mockRequest struct {
  ID     int    `json:"id"`
  Method string `json:"method"`
  Nonce  string `json:"nonce"`
  Params map[string]interface{}
}

type testingFunc func(client *Client)

func TestFormat(t *testing.T) {
  client := New("", "", "test", "test")

  markets := []string{"ETH_BTC", "ETH_COV", "XRP_BTC"}
  expected := []string{"trade.ETH_BTC", "trade.ETH_COV", "trade.XRP_BTC"}

  result := client.format(markets, func(s string) string {
    return fmt.Sprintf("trade.%s", s)
  })

  assert.Equal(t, result, expected)
}

func testSubscribe(t *testing.T, expected string, isPrivate bool, testFunc testingFunc) {
  // prepare expected
  var expectedResponse mockRequest
  err := json.Unmarshal([]byte(expected), &expectedResponse)
  if err != nil {
    t.Fatal("error on parse expected")
  }

  // prepare mock
  client := New("test", "test", "test", "test")
  privateWritingMessage := bytes.NewBuffer(nil)
  publicWritingMessage := bytes.NewBuffer(nil)
  client.connectMock(bytes.NewBuffer(nil), bytes.NewBuffer(nil), privateWritingMessage, publicWritingMessage)

  // call test function
  testFunc(client)

  // get response
  var writingMessage mockRequest
  if isPrivate {
    err = json.Unmarshal(privateWritingMessage.Bytes(), &writingMessage)
  } else {
    err = json.Unmarshal(publicWritingMessage.Bytes(), &writingMessage)
  }
  if err != nil {
    t.Fatal("error on parse writing message")
  }

  // assertion
  assert.NotEqual(t, mockRequest{}, writingMessage)
  // doesn't assert on nonce
  assert.Equal(t, expectedResponse.ID, writingMessage.ID)
  assert.Equal(t, expectedResponse.Method, writingMessage.Method)
  assert.Equal(t, expectedResponse.Params, writingMessage.Params)
}

// for test parse json. this case expected and mock is the same thing
func testResponse(t *testing.T, expected string, isPrivate bool) {
  // prepare expected
  var expectedResponse Response
  err := json.Unmarshal([]byte(expected), &expectedResponse)
  if err != nil {
    t.Fatal("error on parse expected")
  }

  // prepare mock
  client := New("test", "test", "test", "test")
  var privateResponse *bytes.Buffer
  var publicResponse *bytes.Buffer

  testID := 9999
  emptyResponse := fmt.Sprintf(`{"id":%d,"method":"subscribe"}`, testID) // Id 9999 is flag for test
  // mock response.
  if isPrivate {
    privateResponse = bytes.NewBufferString(expected)
    publicResponse = bytes.NewBufferString(emptyResponse)
  } else {
    privateResponse = bytes.NewBufferString(emptyResponse)
    publicResponse = bytes.NewBufferString(expected)
  }
  client.connectMock(privateResponse, publicResponse, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

  msgs := client.Listen()

  var response Response
  for m := range msgs {
    if m.Id != testID {
      response = m
      break
    }
  }

  // assertion
  assert.NotEqual(t, Response{}, response)
  assert.Equal(t, expectedResponse, response)
}

func TestPublicOrderBook(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    expected := `{"id":1,"method":"subscribe","nonce":"","params":{"channels":["book.ETH_BTC.10"]}}`
    testSubscribe(t, expected, false, func(client *Client) { client.SubscribePublicOrderBook(10, "ETH_BTC") })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "method": "subscribe",
      "result": {
        "instrument_name": "ETH_CRO",
        "subscription": "book.ETH_CRO.150",
        "channel": "book",
        "depth": 150,
        "data": [
          {
            "bids": [
              [
                11746.488,
                128,
                8
              ]
            ],
            "asks": [
              [
                11747.488,
                201,
                12
              ]
            ],
            "t": 1587523078844
          }
        ]
      }
    }`
    testResponse(t, jsonExpected, false)
  })
}

func TestPublicTrades(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    expected := `{"id":1,"method":"subscribe","nonce":"","params":{"channels":["trade.ETH_BTC"]}}`
    testSubscribe(t, expected, false, func(client *Client) { client.SubscribePublicTrades("ETH_BTC") })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "method": "subscribe",
      "result": {
        "instrument_name": "ETH_CRO",
        "subscription": "trade.ETH_CRO",
        "channel": "trade",
        "data": [
          {
            "p": 162.12,
            "q": 11.085,
            "s": "buy",
            "d": 1210447366,
            "t": 1587523078844,
            "dataTime": 0
          }
        ]
      }
    }`
    testResponse(t, jsonExpected, false)
  })
}

func TestPublicTickers(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    expected := `{"id":1,"method":"subscribe","nonce":"","params":{"channels":["ticker.ETH_BTC"]}}`
    testSubscribe(t, expected, false, func(client *Client) { client.SubscribePublicTickers("ETH_BTC") })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "method": "subscribe",
      "result": {
        "instrument_name": "ETH_CRO",
        "subscription": "ticker.ETH_CRO",
        "channel": "ticker",
        "data": [
          {
            "h": 1,
            "v": 10232.26315789,
            "a": 173.60263169,
            "l": 0.01,
            "b": 0.01,
            "k": 1.12345680,
            "c": -0.44564773,
            "t": 1587523078844
          }
        ]
      }
    }`
    testResponse(t, jsonExpected, false)
  })
}

func TestSubscribePrivateOrders(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    expected := `{"id":1,"method":"subscribe","nonce":"","params":{"channels":["user.order.ETH_BTC"]}}`
    testSubscribe(t, expected, true, func(client *Client) { client.SubscribePrivateOrders("ETH_BTC") })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "method": "subscribe",
      "result": {
        "instrument_name": "ETH_CRO",
        "subscription": "user.order.ETH_CRO",
        "channel": "user.order",
        "data": [
          {
            "status": "ACTIVE",
            "side": "BUY",
            "price": 1,
            "quantity": 1,
            "order_id": "366455245775097673",
            "client_oid": "my_order_0002",
            "create_time": 1588758017375,
            "update_time": 1588758017411,
            "type": "LIMIT",
            "instrument_name": "ETH_CRO",
            "cumulative_quantity": 0,
            "cumulative_value": 0,
            "avg_price": 0,
            "fee_currency": "CRO",
            "time_in_force":"GOOD_TILL_CANCEL"
          }
        ],
        "channel": "user.order.ETH_CRO"
      }
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestSubscribePrivateTrades(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    expected := `{"id":1,"method":"subscribe","nonce":"","params":{"channels":["user.trade.ETH_BTC"]}}`
    testSubscribe(t, expected, true, func(client *Client) { client.SubscribePrivateTrades("ETH_BTC") })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "method": "subscribe",
      "code": 0,
      "result": {
        "channel": "user.trade",
        "data": [
          {
            "client_oid": "367a0bd0-5033-43b6-8541-7333cb7d2257",
            "create_time": 1612254700593,
            "fee": 0.0003,
            "fee_currency": "ETH",
            "instrument_name": "ETH_CRO",
            "liquidity_indicator": "MAKER",
            "order_id": "1154873104072854624",
            "side": "BUY",
            "trade_id": "1154873105026644514",
            "traded_price": 0.5,
            "traded_quantity": 0.3
          }
        ],
        "instrument_name": "ETH_CRO",
        "subscription": "user.trade.ETH_CRO"
      }
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestSubscribePrivateBalanceUpdates(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    expected := `{"id":1,"method":"subscribe","nonce":"","params":{"channels":["user.balance"]}}`
    testSubscribe(t, expected, true, func(client *Client) { client.SubscribePrivateBalanceUpdates() })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "method": "subscribe",
      "result": {
        "subscription": "user.balance",
        "channel": "user.balance",
        "data": [
          {
            "currency": "CRO",
            "balance": 99999999947.99626,
            "available": 99999988201.50826,
            "order": 11746.488,
            "stake": 0
          }
        ],
        "channel": "user.balance"
      }
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestCreateOrder(t *testing.T) {
  t.Run("Subscribe BUY Market", func(t *testing.T) {
    // prepare expected
    uuid := uuid.New()
    price := decimal.NewFromFloat(0.01)
    volume := decimal.NewFromFloat(0.0001)
    orderType := "MARKET"

    expected := fmt.Sprintf(
      `{"id":1,"method":"private/create-order","nonce":"","params":{"client_oid":"%s","instrument_name":"ETH_CRO","price":"%s","notional":"%s","side":"%s","type":"%s"}}`,
      uuid, price.String(), volume.String(), "BUY", orderType,
    )
    testSubscribe(t, expected, true, func(client *Client) {
      client.CreateOrder(
        1,
        "ETH",
        "CRO",
        "buy",
        orderType,
        price,
        volume,
        uuid,
      )
    })
  })

  t.Run("Subscribe BUY Limit", func(t *testing.T) {
    // prepare expected
    uuid := uuid.New()
    price := decimal.NewFromFloat(0.01)
    volume := decimal.NewFromFloat(0.0001)
    orderType := "LIMIT"

    expected := fmt.Sprintf(
      `{"id":1,"method":"private/create-order","nonce":"","params":{"client_oid":"%s","instrument_name":"ETH_CRO","price":"%s","quantity":"%s","side":"%s","type":"%s"}}`,
      uuid, price.String(), volume.String(), "BUY", orderType,
    )
    testSubscribe(t, expected, true, func(client *Client) {
      client.CreateOrder(
        1,
        "ETH",
        "CRO",
        "buy",
        orderType,
        price,
        volume,
        uuid,
      )
    })
  })

  t.Run("Subscribe SELL Limit", func(t *testing.T) {
    // prepare expected
    uuid := uuid.New()
    price := decimal.NewFromFloat(0.01)
    volume := decimal.NewFromFloat(0.0001)
    orderType := "LIMIT"

    expected := fmt.Sprintf(
      `{"id":1,"method":"private/create-order","nonce":"","params":{"client_oid":"%s","instrument_name":"ETH_CRO","price":"%s","quantity":"%s","side":"%s","type":"%s"}}`,
      uuid, price.String(), volume.String(), "SELL", orderType,
    )
    testSubscribe(t, expected, true, func(client *Client) {
      client.CreateOrder(
        1,
        "ETH",
        "CRO",
        "sell",
        orderType,
        price,
        volume,
        uuid,
      )
    })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "id": 11,
      "method": "private/create-order",
      "result": {
        "order_id": "337843775021233500",
        "client_oid": "my_order_0002"
      }
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestCancelOrder(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    remoteID := sql.NullString{String: "1138210129647637539", Valid: true}

    // prepare expected
    expected := fmt.Sprintf(
      `{"id":1,"method":"private/cancel-order","nonce":"","params":{"instrument_name":"ETH_CRO","order_id":"%s"}}`,
      remoteID.String,
    )
    testSubscribe(t, expected, true, func(client *Client) {
      client.CancelOrder(
        1,
        remoteID.String,
        "ETH_CRO",
      )
    })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "id": 11,
      "method": "private/cancel-order",
      "code":0
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestCancelAllOrders(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    expected := `{"id":1,"method":"private/cancel-all-orders","nonce":"","params":{"instrument_name":"ETH_CRO"}}`
    testSubscribe(t, expected, true, func(client *Client) { client.CancelAllOrders(1, "ETH_CRO") })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "id": 12,
      "method": "private/cancel-all-order",
      "code": 0
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestGetOrderDetails(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare expected
    remoteID := sql.NullString{String: "1138210129647637539", Valid: true}
    expected := `{"id":1,"method":"private/get-order-detail","nonce":"","params":{"order_id":"1138210129647637539"}}`
    testSubscribe(t, expected, true, func(client *Client) { client.GetOrderDetails(1, remoteID) })
  })

  t.Run("Read response", func(t *testing.T) {
    jsonExpected := `{
      "id": 11,
      "method": "private/get-order-detail",
      "code": 0,
      "result": {
        "trade_list": [
          {
            "side": "BUY",
            "instrument_name": "ETH_CRO",
            "fee": 0.007,
            "trade_id": "371303044218155296",
            "create_time": 1588902493045,
            "traded_price": 7,
            "traded_quantity": 7,
            "fee_currency": "CRO",
            "order_id": "371302913889488619"
          }
        ],
        "order_info": {
          "status": "FILLED",
          "side": "BUY",
          "order_id": "371302913889488619",
          "client_oid": "9_yMYJDNEeqHxLqtD_2j3g",
          "create_time": 1588902489144,
          "update_time": 1588902493024,
          "type": "LIMIT",
          "instrument_name": "ETH_CRO",
          "cumulative_quantity": 7,
          "cumulative_value": 7,
          "avg_price": 7,
          "fee_currency": "CRO",
          "time_in_force": "GOOD_TILL_CANCEL",
          "exec_inst": "POST_ONLY"
        }
      }
    }`
    testResponse(t, jsonExpected, true)
  })
}

func TestRespondHeartBeat(t *testing.T) {
  t.Run("Subscribe", func(t *testing.T) {
    // prepare mock
    client := New("test", "test", "test", "test")
    privateWritingMessage := bytes.NewBuffer(nil)
    publicWritingMessage := bytes.NewBuffer(nil)
    client.connectMock(bytes.NewBuffer(nil), bytes.NewBuffer(nil), privateWritingMessage, publicWritingMessage)

    t.Run("private", func(t *testing.T) {
      var writingMessage mockRequest
      var expectedResponse mockRequest
      expected := `{"id":1,"method":"public/respond-heartbeat"}`

      // start test
      client.respondHeartBeat(true, 1)
      json.Unmarshal(privateWritingMessage.Bytes(), &writingMessage)
      // prepare expected
      json.Unmarshal([]byte(expected), &expectedResponse)

      assert.NotEqual(t, mockRequest{}, writingMessage)
      assert.Equal(t, expectedResponse, writingMessage)
    })

    t.Run("public", func(t *testing.T) {
      var writingMessage mockRequest
      var expectedResponse mockRequest
      expected := `{"id":1,"method":"public/respond-heartbeat"}`

      // start test
      client.respondHeartBeat(false, 1)
      json.Unmarshal(publicWritingMessage.Bytes(), &writingMessage)
      // prepare expected
      json.Unmarshal([]byte(expected), &expectedResponse)

      assert.NotEqual(t, mockRequest{}, writingMessage)
      assert.Equal(t, expectedResponse, writingMessage)
    })
  })
}
