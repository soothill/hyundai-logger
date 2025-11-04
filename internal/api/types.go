package api

import "time"

// VehiclesResponse represents the response from the vehicles endpoint
type VehiclesResponse struct {
	Vehicles []Vehicle `json:"resMsg,omitempty"`
	Result   []Vehicle `json:"result,omitempty"` // Alternative field name for some regions
}

// Vehicle represents a vehicle in the account
type Vehicle struct {
	VehicleID    string `json:"vehicleId"`
	VIN          string `json:"vin"`
	Nickname     string `json:"nickname"`
	VehicleName  string `json:"vehicleName"`
	VehicleModel string `json:"vehicleModel"`
	Year         string `json:"year"`
	Color        string `json:"color"`
}

// VehicleStatus represents the complete vehicle status
type VehicleStatus struct {
	VehicleStatus   GeneralStatus  `json:"vehicleStatus"`
	VehicleLocation Location       `json:"vehicleLocation"`
	OdometerStatus  OdometerStatus `json:"odometer"`
	EVStatus        *EVStatus      `json:"evStatus,omitempty"`
	Climate         ClimateStatus  `json:"airCtrlStatus"`
	DoorStatus      DoorStatus     `json:"doorOpen"`
	TireStatus      TireStatus     `json:"tirePressure"`
	LastUpdateTime  time.Time      `json:"lastUpdateTime"`
}

// GeneralStatus represents general vehicle status
type GeneralStatus struct {
	Engine            bool        `json:"engine"`
	Locked            bool        `json:"doorLock"`
	LowFuelLight      bool        `json:"lowFuelLight"`
	AirCondition      bool        `json:"airCtrl"`
	DefrostStatus     string      `json:"defrost"`
	TrunkOpen         bool        `json:"trunkOpen"`
	HoodOpen          bool        `json:"hoodOpen"`
	FuelLevel         int         `json:"fuelLevel"`
	BatteryVoltage    float64     `json:"battery"`
	RemoteStartStatus RemoteStart `json:"remoteStart"`
}

// RemoteStart represents remote start status
type RemoteStart struct {
	RemoteStartActive bool      `json:"remoteStartActive"`
	RemoteStartTime   time.Time `json:"remoteStartTime"`
}

// Location represents vehicle location
type Location struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Altitude  float64   `json:"altitude"`
	Speed     float64   `json:"speed"`
	Heading   float64   `json:"heading"`
	Time      time.Time `json:"time"`
}

// OdometerStatus represents odometer information
type OdometerStatus struct {
	Value int    `json:"value"`
	Unit  string `json:"unit"`
}

// EVStatus represents electric vehicle specific status
type EVStatus struct {
	BatteryLevel            int     `json:"batteryLevel"`
	BatteryCapacity         float64 `json:"batteryCapacity"`
	BatteryCharge           bool    `json:"batteryCharge"`
	PluggedIn               bool    `json:"batteryPlugin"`
	ChargingPower           float64 `json:"chargingPower"`
	EstimatedChargeTime     int     `json:"remainTime"`
	TargetChargeLevel       int     `json:"targetChargeLevel"`
	ChargeTargetSOC         int     `json:"chargeTargetSOC"`
	RangeEV                 float64 `json:"drvDistance"`
	ChargingCurrent         float64 `json:"chargingCurrent"`
	ChargingVoltage         float64 `json:"chargingVoltage"`
	ChargeMode              string  `json:"chargeMode"`
	ChargeStatus            string  `json:"chargeStatus"`
	EstimatedFullChargeTime int     `json:"estimatedFullChargeTime"`
	EstimatedPortableTime   int     `json:"estimatedPortableTime"`
	EstimatedStationTime    int     `json:"estimatedStationTime"`
}

// ClimateStatus represents climate control status
type ClimateStatus struct {
	Active         bool    `json:"airCtrl"`
	InteriorTemp   float64 `json:"airTemp"`
	ExteriorTemp   float64 `json:"outTemp"`
	TargetTemp     float64 `json:"airCtrlTempValue"`
	FanSpeed       int     `json:"airCtrlFanSpeed"`
	DefrostActive  bool    `json:"defrost"`
	RearDefrost    bool    `json:"rearBlast"`
	SteeringWheel  bool    `json:"steerWheelHeat"`
	SideMirrorHeat bool    `json:"sideMirrorHeat"`
	SeatHeatLeft   int     `json:"seatHeatLeft"`
	SeatHeatRight  int     `json:"seatHeatRight"`
}

// DoorStatus represents door and lock status
type DoorStatus struct {
	FrontLeft  bool `json:"frontLeft"`
	FrontRight bool `json:"frontRight"`
	BackLeft   bool `json:"backLeft"`
	BackRight  bool `json:"backRight"`
	Trunk      bool `json:"trunk"`
	Hood       bool `json:"hood"`
}

// TireStatus represents tire pressure information
type TireStatus struct {
	FrontLeftPSI     float64 `json:"frontLeftPsi"`
	FrontRightPSI    float64 `json:"frontRightPsi"`
	RearLeftPSI      float64 `json:"rearLeftPsi"`
	RearRightPSI     float64 `json:"rearRightPsi"`
	FrontLeftStatus  string  `json:"frontLeftStatus"`
	FrontRightStatus string  `json:"frontRightStatus"`
	RearLeftStatus   string  `json:"rearLeftStatus"`
	RearRightStatus  string  `json:"rearRightStatus"`
}

// CommandResponse represents response from control commands
type CommandResponse struct {
	Status    string `json:"status"`
	ErrorCode string `json:"errorCode,omitempty"`
	ErrorMsg  string `json:"errorMsg,omitempty"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}
