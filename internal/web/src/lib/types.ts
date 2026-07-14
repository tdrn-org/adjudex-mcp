// Mirror Go domain types (json:"snake_case" tags)
// See: internal/domain/portfolio.go, internal/domain/quote.go

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
  symbol: string;
  currency: string;
  quantity: number;
  entry_price: number;
  entry_date: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface Holding {
  id: string;
  symbol: string;
  currency: string;
  quantity: number;
  entry_price: number;
  entry_date: string;
  notes?: string;
  created_at: string;
  updated_at: string;
  current_price: number;
  market_value: number;
  pnl: number;
  pnl_percent: number;
}

export interface Quote {
  symbol: string;
  timestamp: string;
  currency: string;
  open: number;
  high: number;
  low: number;
  close: number;
  price: number;
  volume: number;
  source: string;
  source_timestamp: string;
}

export interface PriceHistory {
  symbol: string;
  quotes: Quote[];
}

export interface ServerInfo {
  version: string;
}
