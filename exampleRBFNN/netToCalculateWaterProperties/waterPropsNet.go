// cgo LDFLAGS:
//    Linux:  -L/usr/lib/ -lseuif97 -lm
//    Windows: -LC:/Windows/system -llibseuif97
// go build -o demo.exe demo.go
package main

/*
#cgo LDFLAGS: -LC:/Windows/system -llibseuif97
#include "seuif97.h"
*/
import "C"
import "fmt"

const (
	oidPressure = iota
	oidTemperature
	oidDensity
	oidSpecificVolume
	oidEnthalpy
	oidEntropy
	oidExergy
	oidSpecificInternalEnergy
	oidSpecificIsobaricHeatCapacity
	oidSpecificIsochoricHeatCapacity
	oidSpeedOfSound
	oidIsentropicExponent
	oidSpecificHelmholtzFreeEnergy       //kJ/kg	f	12
	oidSpecificGibbsFreeEnergy           //kJ/kg	g	13
	oidCompressibilityFactor             //z	14
	oidSteamQuality                      //x	15
	oidRegion                            //r	16
	oidIsobaricCubicExpansionCoefficient //1/K	αv	17
	oidIsothermalCompressibility         //1/MPa	kT	18
	oiddVdTp                             // Partial derivative (∂V/∂T)p	m^3/(kg·K)	(∂V/∂T)p	19
	oiddVdPT                             // Partial derivative (∂V/∂P)T	m^3/(kg·MPa)	(∂V/∂P)T	20
	oiddpdtv                             // Partial derivative (∂p/∂t)v	MPa/K	(∂p/∂t)v	21
	oidIsothermalThrottlingCoefficient   //kJ/(kg·MPa)	δt	22
	oidJouleThomsonCoefficient           //K/MPa	μ	23
	oidDynamicViscosity                  //kg/(m·s)	η	24
	oidKinematicViscosity                //m²/s	ν	25
	oidThermalConductivity               //W/(m·K)	λ	26
	oidThermalDiffusivity                //µm²/s	a	27
	oidPrandtlNumber                     //Pr	28
	oidSurfaceTension
)
const (
	OidP     = oidPressure
	OidT     = oidTemperature
	Oidρ     = oidDensity
	Oidv     = oidSpecificVolume
	Oidh     = oidEnthalpy
	Oids     = oidEntropy
	Oide     = oidExergy
	Oidu     = oidSpecificInternalEnergy
	OidCp    = oidSpecificIsobaricHeatCapacity
	OidCv    = oidSpecificIsochoricHeatCapacity
	OidW     = oidSpeedOfSound
	Oidk     = oidIsentropicExponent
	Oidf     = oidSpecificHelmholtzFreeEnergy
	Oidg     = oidSpecificGibbsFreeEnergy
	OidG     = oidCompressibilityFactor
	Oidx     = oidSteamQuality
	Oidr     = oidRegion
	Oidav    = oidIsobaricCubicExpansionCoefficient
	OidkT    = oidIsothermalCompressibility
	Oiddvdtp = oiddVdTp
	Oiddvdpt = oiddVdPT
	Oiddpdtv = oiddpdtv
	Oidδt    = oidIsothermalThrottlingCoefficient //kJ/(kg·MPa)	δt	22
	Oidμ     = oidJouleThomsonCoefficient         //K/MPa	μ	23
	Oidn     = oidDynamicViscosity                //kg/(m·s)	η	24
	Oidν     = oidKinematicViscosity              //m²/s	ν	25
	Oidλ     = oidThermalConductivity             //W/(m·K)	λ	26
	Oida     = oidThermalDiffusivity              //µm²/s	a	27
	OidPr    = oidPrandtlNumber                   //Pr	28
	Oidγ     = oidSurfaceTension
)

// TODO: example using https://github.com/thermalogic/SEUIF97
func main() {
	p := 16.14
	t := 512.0
	oid := 4 // See https://github.com/thermalogic/SEUIF97#properties
	h := C.pt(C.double(p), C.double(t), C.int(oid))
	fmt.Printf("(p,t)=(%.2f, %.2f), h=%.2f\n", p, t, float64(h))
}
func getThermoProp(typ string, a, b float64, oid int) float64 {
	switch typ {
	case "pt":
		return C.pt(C.double(a), C.double(b), C.int(oid))
	case "ph":
		return C.ph(C.double(a), C.double(b), C.int(oid))
	case "ps":
		return C.ps(C.double(a), C.double(b), C.int(oid))
	case "pv":
		return C.pv(C.double(a), C.double(b), C.int(oid))
	case "hs":
		return C.hs(C.double(a), C.double(b), C.int(oid))
	case "th":
		return C.th(C.double(a), C.double(b), C.int(oid))
	case "ts":
		return C.ts(C.double(a), C.double(b), C.int(oid))
	case "tv":
		return C.tv(C.double(a), C.double(b), C.int(oid))
	case "hx":
		return C.hx(C.double(a), C.double(b), C.int(oid))
	case "tx":
		return C.tx(C.double(a), C.double(b), C.int(oid))
	case "vx":
		return C.vx(C.double(a), C.double(b), C.int(oid))
	case "sx":
		return C.sx(C.double(a), C.double(b), C.int(oid))
	default:
		panic("invalid typ arg")
	}
}

// TODO: https://github.com/thermalogic/SEUIF97 ?

/* TODO: Saturation calculator: given T OR P.
density (kg/m3),
specific volume (m3/kg) vf, vg vfg can be calculated
specific enthalpy hf, hg (kJ/kg) hfg can be calculated
internal energy uf, ug
entropy sf, sg
specific heat Cp (kJ/kg K)
viscosity(s?), PROBABLY NOT!

*/
/* TODO: Given T, P. Return:
Whether or not Saturated...
density (kg/m3),
specific volume (m3/kg) vf, vg vfg can be calculated
specific enthalpy hf, hg (kJ/kg) hfg can be calculated
internal energy uf, ug
entropy sf, sg
specific heat Cp (kJ/kg K)
viscosity(s?), PROBABLY NOT!

*/

/* TODO: Given T
calculate Psat and all saturation properties
// TODO: https://www.nist.gov/system/files/documents/srd/NISTIR5078-Tab1.pdf
*/

/* TODO: Given P
calculate Tsat and all saturation properties
// TODO: https://www.nist.gov/system/files/documents/srd/NISTIR5078-Tab2.pdf
*/

/* Saturation properties:
rhof, rhog
vf, vg
hf, hg,
uf, ug,
sf, sg
specific heat Cp (kJ/kg K)?
*/
