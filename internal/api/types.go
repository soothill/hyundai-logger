// Copyright (c) 2025 Darren Soothill
// Email: darren [at] soothill [dot] com
// Licensed under the MIT License
//

package api

import "time"

// AuthResponse represents the authentication response
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// VehiclesResponse represents the response from the vehicles endpoint
type VehiclesResponse struct {
	Vehicles []Vehicle `json:"vehicles"`
}

// Vehicle represents a vehicle in the account
type Vehicle struct {
	VehicleID   string `json:"vehicleId"`
	VIN         string `json:"vin"`
	Nickname    string `json:"nickname"`
	Year        int    `json:"year"`
	Make        string `json:"make"`
	Model       string `json:"model"`
	Color       string `json:"color"`
	Generation  string `json:"generation"`
	RegisteredDate time.Time `json:"registeredDate"`
}

// VehicleStatus represents the complete status of a vehicle
type VehicleStatus struct {
	Timestamp         time.Time         `json:"timestamp"`
	VIN               string            `json:"vin"`
	Engine            EngineStatus      `json:"engine"`
	Climate           ClimateStatus     `json:"climate"`
	Doors             DoorsStatus       `json:"doors"`
	Battery           BatteryStatus     `json:"battery"`
	Tire              TireStatus        `json:"tire"`
	EV                *EVStatus         `json:"ev,omitempty"`
	Location          *LocationData     `json:"location,omitempty"`
	Odometer          float64           `json:"odometer"`
	FuelLevel         float64           `json:"fuelLevel"`
	DefrostStatus     bool              `json:"defrost"`
	SteeringWheelHeat bool              `json:"steeringWheelHeat"`
	SideMirrorHeat    bool              `json:"sideMirrorHeat"`
	RearWindowHeat    bool              `json:"rearWindowHeat"`
	Washer            WasherStatus      `json:"washer"`
}

// EngineStatus represents engine information
type EngineStatus struct {
	Running          bool    `json:"running"`
	RemoteStartState bool    `json:"remoteStartState"`
	Rpm              int     `json:"rpm"`
	RangeKM          float64 `json:"rangeKm"`
	RangeMiles       float64 `json:"rangeMiles"`
}

// ClimateStatus represents climate control information
type ClimateStatus struct {
	Active           bool    `json:"active"`
	TargetTemp       float64 `json:"targetTemp"`
	InteriorTemp     float64 `json:"interiorTemp"`
	ExteriorTemp     float64 `json:"exteriorTemp"`
	AirCondition     bool    `json:"airCondition"`
	Heater           bool    `json:"heater"`
	AutoMode         bool    `json:"autoMode"`
	FanSpeed         int     `json:"fanSpeed"`
}

// DoorsStatus represents door lock status
type DoorsStatus struct {
	Locked          bool `json:"locked"`
	FrontLeft       bool `json:"frontLeft"`
	FrontRight      bool `json:"frontRight"`
	BackLeft        bool `json:"backLeft"`
	BackRight       bool `json:"backRight"`
	Trunk           bool `json:"trunk"`
	Hood            bool `json:"hood"`
}

// BatteryStatus represents 12V battery information
type BatteryStatus struct {
	Level           float64 `json:"level"`
	Voltage         float64 `json:"voltage"`
	ChargeTime      int     `json:"chargeTime"`
	WarningLight    bool    `json:"warningLight"`
}

// TireStatus represents tire pressure information
type TireStatus struct {
	FrontLeft       TirePressure `json:"frontLeft"`
	FrontRight      TirePressure `json:"frontRight"`
	RearLeft        TirePressure `json:"rearLeft"`
	RearRight       TirePressure `json:"rearRight"`
	WarningLight    bool         `json:"warningLight"`
}

// TirePressure represents individual tire pressure
type TirePressure struct {
	PSI        float64 `json:"psi"`
	Status     string  `json:"status"` // normal, low, high
}

// EVStatus represents electric vehicle specific information
type EVStatus struct {
	BatteryLevel         float64   `json:"batteryLevel"`
	BatteryCapacity      float64   `json:"batteryCapacity"`
	Charging             bool      `json:"charging"`
	ChargingPower        float64   `json:"chargingPower"`
	EstimatedCurrentCharge int     `json:"estimatedCurrentCharge"`
	EstimatedFullCharge  int       `json:"estimatedFullCharge"`
	RangeKM              float64   `json:"rangeKm"`
	RangeMiles           float64   `json:"rangeMiles"`
	PluggedIn            bool      `json:"pluggedIn"`
	ChargeTargetPercent  int       `json:"chargeTargetPercent"`
	ChargeEndTime        time.Time `json:"chargeEndTime"`
}

// WasherStatus represents washer fluid status
type WasherStatus struct {
	Level        string `json:"level"` // full, good, low, empty
	WarningLight bool   `json:"warningLight"`
}

// Location represents vehicle location
type Location struct {
	Timestamp time.Time     `json:"timestamp"`
	VIN       string        `json:"vin"`
	Location  LocationData  `json:"location"`
}

// LocationData represents GPS coordinates
type LocationData struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
	Speed     float64 `json:"speed"`
	Heading   float64 `json:"heading"`
}

// Odometer represents odometer reading
type Odometer struct {
	Timestamp time.Time `json:"timestamp"`
	VIN       string    `json:"vin"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"` // km or miles
}
