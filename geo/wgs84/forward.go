// Copyright 2011 The Avalon Project Authors. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the LICENSE file.
//
// This is a translation of a part of GeographicLib-1.15 to Go.
//
// Original copyright notice: 
// Copyright (c) Charles Karney (2011) <charles@karney.com> and licensed
// under the MIT/X11 License.  For more information, see
//     http://geographiclib.sourceforge.net/
//
// The original license is in LICENSE-GeographicLib.txt
//

package wgs84

import "math"

// Compute the Point (lat2, lon2) obtained by following the geodesic
// line from (lat1,lon2) in the direction azi1 over a distance of s12
// [meters] and the arriving azimuth azi2.  All angles are in radians.
// Azimuths are clockwise from the North.  Positive latitudes are North,
// positive longitudes are East. Unlike the C++ original, azi2 points
// in the incoming direction.
func Forward(lat1, lon1, azi1, s12 float64) (lat2, lon2, azi2 float64) {
	return NewGeodesicLine(lat1, lon1, azi1).Position(s12)
}

// A GeodesicLine represents a geodesic around the ellipsoid based in some point, under some azimuth.
type GeodesicLine struct {
	lat1, lon1, azi1                                       float64
	salp0, calp0, k2                                       float64
	salp1, calp1, ssig1, csig1, stau1, ctau1, somg1, comg1 float64
	_A1m1, _A2m1, _A3c, _B11, _B21, _B31, _A4, _B41        float64
	// index zero elements of C1a, C1pa, C2a, C3a are unused, 
	// all the elements of C4a are used
	_C1a  [_nC1 + 1]float64
	_C1pa [_nC1p + 1]float64
	_C2a  [_nC2 + 1]float64
	_C3a  [_nC3]float64
	_C4a  [_nC4]float64
}

// Construct an object representing a geodesic line
// through (lat1, lon1) at azimuth azi1.  All angles are in radians.
// Azimuths are clockwise from the North.  Positive latitudes are North,
// positive longitudes are East.
func NewGeodesicLine(lat1, lon1, azi1 float64) *GeodesicLine {

	g := new(GeodesicLine)

	g.lat1, g.lon1, g.azi1 = lat1, angNormalize(lon1), angRound(angNormalize(azi1))
	g.salp1, g.calp1 = math.Sincos(azi1)

	sbet1, cbet1 := math.Sincos(lat1)
	sbet1, cbet1 = sinCosNorm(_f1*sbet1, cbet1)

	g.salp0 = g.salp1 * cbet1
	g.calp0 = math.Hypot(g.calp1, g.salp1*sbet1)

	g.ssig1, g.csig1 = sbet1, 1.
	g.somg1, g.comg1 = g.salp0*sbet1, 1.
	if sbet1 != 0 || g.calp1 != 0 {
		g.csig1 = cbet1 * g.calp1
		g.comg1 = g.csig1
	}
	g.ssig1, g.csig1 = sinCosNorm(g.ssig1, g.csig1)
	g.somg1, g.comg1 = sinCosNorm(g.somg1, g.comg1)

	g.k2 = g.calp0 * g.calp0 * _ep2
	eps := g.k2 / (2*(1+math.Sqrt(1+g.k2)) + g.k2)

	g._A1m1 = a1m1f(eps)
	c1f(eps, g._C1a[:])
	g._B11 = sinSeries(g.ssig1, g.csig1, g._C1a[:], _nC1)
	s, c := math.Sincos(g._B11)

	g.stau1 = g.ssig1*c + g.csig1*s
	g.ctau1 = g.csig1*c - g.ssig1*s

	c1pf(eps, g._C1pa[:])

	c3f(eps, g._C3a[:])
	g._A3c = -_f * g.salp0 * a3f(eps)
	g._B31 = sinSeries(g.ssig1, g.csig1, g._C3a[:], _nC3-1)

	return g
}

// Compute the Point (lat2,lon2) obtained by following the geodesic
// line from it's basepoint over a distance of s12 [meters] and the
// arriving azimuth azi2.  All angles are in radians.
// Azimuths are clockwise from the North.  Positive latitudes are North,
// positive longitudes are East.  Unlike the C++ original, azi2 points in
// the incoming direction.
func (g *GeodesicLine) Position(s12 float64) (lat2, lon2, azi2 float64) {

	// Note: omitted calls to angRound that were in the C++ original
	var sig12, ssig12, csig12, B12 float64

	tau12 := s12 / (_b * (1 + g._A1m1))
	s, c := math.Sincos(tau12)
	// tau2 = tau1 + tau12
	B12 = -sinSeries(g.stau1*c+g.ctau1*s, g.ctau1*c-g.stau1*s, g._C1pa[:], _nC1p)
	sig12 = tau12 - (B12 - g._B11)
	ssig12, csig12 = math.Sincos(sig12)

	var omg12, lam12, lon12 float64
	var ssig2, csig2, sbet2, cbet2, somg2, comg2, salp2, calp2 float64
	// sig2 = sig1 + sig12
	ssig2 = g.ssig1*csig12 + g.csig1*ssig12
	csig2 = g.csig1*csig12 - g.ssig1*ssig12

	// sin(bet2) = cos(alp0) * sin(sig2)
	sbet2 = g.calp0 * ssig2
	// Alt: cbet2 = hypot(csig2, salp0 * ssig2);
	cbet2 = math.Hypot(g.salp0, g.calp0*csig2)
	if cbet2 == 0 { // I.e., salp0 = 0, csig2 = 0.  Break the degeneracy in this case
		csig2 = _tiny
		cbet2 = csig2
	}

	// tan(omg2) = sin(alp0) * tan(sig2)
	somg2 = g.salp0 * ssig2
	comg2 = csig2 // No need to normalize
	// tan(alp0) = cos(sig2)*tan(alp2)
	salp2 = g.salp0
	calp2 = g.calp0 * csig2 // No need to normalize
	// omg12 = omg2 - omg1
	omg12 = math.Atan2(somg2*g.comg1-comg2*g.somg1, comg2*g.comg1+somg2*g.somg1)

	lam12 = omg12 + g._A3c*(sig12+(sinSeries(ssig2, csig2, g._C3a[:], _nC3-1)-g._B31))
	lon12 = angNormalize(lam12)
	// Can't use AngNormalize because longitude might have wrapped multiple times.

	lon2 = angNormalize(g.lon1 + lon12)

	lat2 = math.Atan2(sbet2, _f1*cbet2)

	// minus signs give range [-180, 180). 0- converts -0 to +0.
	azi2 = 0 - math.Atan2(salp2, -calp2) // reversed sign so it points backwards

	return
}
