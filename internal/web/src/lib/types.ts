// Mirror Go domain types (json:"snake_case" tags)
// See: internal/domain/portfolio.go, internal/domain/money.go, internal/domain/quote.go
//
// Currency discipline: a monetary value never travels as a bare number. Amounts
// carry their currency (Amount), prices carry their currency AND provenance
// (ListingPrice). A value without a currency would be ambiguous — the same
// instrument can be booked in EUR and quoted in USD.

export interface Portfolio {
  id: string;
  name: string;
  description: string;
  positions: Position[] | null;
  created_at: string;
  updated_at: string;
}

export interface Position {
  id: string;
  /** Technical identity string of the listing, e.g. "twelvedata:isin=US0378331005". */
  listing: string;
  /** Currency the entry price is denominated in — fixed once booked. */
  currency: string;
  quantity: number;
  entry_price: number;
  entry_date: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

/** A monetary value together with the currency it is denominated in. */
export interface Amount {
  currency: string;
  value: number;
}

/** A price as reported by a provider, including its provenance. */
export interface ListingPrice {
  listing: string;
  timestamp: string;
  currency: string;
  price: number;
  source: string;
  source_timestamp: string;
}

export interface Holding extends Position {
  /** The quote this holding is valued with, in its ORIGINAL currency. */
  current_price: ListingPrice;
  /** quantity x entry_price, in the position currency. */
  entry_value: Amount;
  /** quantity x current_price, converted to the requested display currency. */
  market_value: Amount;
  /**
   * PnL is only present when entry and market value could be brought into the
   * same currency. Absent means "not computable", not "zero" — show a dash.
   */
  pnl?: Amount;
  pnl_percent?: number;
}

export interface Quote extends ListingPrice {
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface PriceHistory {
  listing: string;
  quotes: Quote[];
}

/** An exchange rate used to convert one currency into another. */
export interface ExchangeRate {
  timestamp: string;
  base: string;
  quote: string;
  rate: number;
  source: string;
  source_timestamp: string;
}

export interface ServerInfo {
  version: string;
}

export interface PortfolioSummary {
  id: string;
  name: string;
  /** Currency this summary is expressed in. */
  currency: string;
  market_value: number;
  pnl: number;
  pnl_percent: number;
  /** Total number of positions in the portfolio. */
  position_count: number;
  /**
   * Positions that actually contributed to the total. Lower than position_count
   * means some quotes could not be resolved and the total is incomplete.
   */
  priced_count: number;
  /** Exchange rates applied to build this summary. */
  rates?: ExchangeRate[];
}
