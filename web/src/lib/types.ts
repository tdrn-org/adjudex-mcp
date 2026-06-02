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
	quantity: number;
	entry_price: number;
	entry_date: string;
	notes: string;
	created_at: string;
	updated_at: string;
}

export interface Holding {
	position: Position;
	current_price: number;
	market_value: number;
	pnl: number;
	pnl_pct: number;
}

export interface Quote {
	symbol: string;
	timestamp: string;
	open: number;
	high: number;
	low: number;
	close: number;
	volume: number;
	source: string;
}

export interface PriceHistory {
	symbol: string;
	quotes: Quote[];
}

export interface IndicatorValue {
	symbol: string;
	type: string;
	period: number;
	value: number;
	timestamp: string;
}
