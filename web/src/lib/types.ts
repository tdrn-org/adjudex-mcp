export interface Portfolio {
	ID: string;
	Name: string;
	Description: string;
	Positions: Position[] | null;
	CreatedAt: string;
	UpdatedAt: string;
}

export interface Position {
	ID: string;
	Symbol: string;
	Quantity: number;
	EntryPrice: number;
	EntryDate: string;
	Notes: string;
}

export interface Holding {
	Position: Position;
	CurrentPrice: number;
	MarketValue: number;
	PnL: number;
	PnLPercent: number;
}

export interface Quote {
	Symbol: string;
	Timestamp: string;
	Open: number;
	High: number;
	Low: number;
	Close: number;
	Volume: number;
	Source: string;
}

export interface PriceHistory {
	Symbol: string;
	Quotes: Quote[];
}

export interface IndicatorValue {
	Symbol: string;
	Type: string;
	Period: number;
	Value: number;
	Timestamp: string;
}
