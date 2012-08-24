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

import (
	"math"
)

func sg(x bool) float64 {
	if x { return 1 }
	return -1
}

// Given two points (lat1, lon1) and (lat2, lon2), compute a geodesic
// between them, and return the distance in meters and the azimuths in
// both points.  All angles are in radians.  Azimuths are clockwise from the north.
// Positive latitudes are North, positive Longitudes are East.
// Unlike the C++ original, azi2 points in the incoming direction.
func Inverse(lat1, lon1, lat2, lon2 float64) (s12, azi1, azi2 float64) {
	lon12 := angNormalize(lon2 - lon1)
	lon12 = angRound(lon12)
	// Make longitude difference positive.
	lonsign := sg(lon12 >= 0)
	lon12 *= lonsign
	if lon12 == math.Pi {
		lonsign = 1
	}

	// If really close to the equator, treat as on equator.
	lat1 = angRound(lat1)
	lat2 = angRound(lat2)

	// Swap points so that point with higher (abs) latitude is point 1
	swapp := sg(math.Abs(lat1) >= math.Abs(lat2))
	if swapp < 0 {
		lonsign *= -1
		lat1, lat2 = lat2, lat1
	}

	// Make lat1 <= 0
	latsign := sg(lat1 < 0)
	lat1 *= latsign
	lat2 *= latsign

	// Now we have
	//
	//     0 <= lon12 <= 180
	//     -90 <= lat1 <= 0
	//     lat1 <= lat2 <= -lat1
	//
	// lonsign, swapp, latsign register the transformation to bring the
	// coordinates to this canonical form.  In all cases, false means no change was
	// made.  We make these transformations so that there are few cases to
	// check, e.g., on verifying quadrants in atan2.  In addition, this
	// enforces some symmetries in the results returned.

	var phi, sbet1, cbet1, sbet2, cbet2, s12x, m12x float64

	phi = lat1
	// Ensure cbet1 = +epsilon at poles
	sbet1, cbet1 = math.Sincos(phi)
	sbet1 *= _f1
	if cbet1 == 0. && lat1 < 0 {
		cbet1 = _tiny
	}
	sbet1, cbet1 = sinCosNorm(sbet1, cbet1)

	phi = lat2
	// Ensure cbet2 = +epsilon at poles
	sbet2, cbet2 = math.Sincos(phi)
	sbet2 *= _f1
	if cbet2 == 0. {
		cbet2 = _tiny
	}
	sbet2, cbet2 = sinCosNorm(sbet2, cbet2)

	// If cbet1 < -sbet1, then cbet2 - cbet1 is a sensitive measure of the
	// |bet1| - |bet2|.  Alternatively (cbet1 >= -sbet1), abs(sbet2) + sbet1 is
	// a better measure.  This logic is used in assigning calp2 in Lambda12.
	// Sometimes these quantities vanish and in that case we force bet2 = +/-
	// bet1 exactly.  An example where is is necessary is the inverse problem
	// 48.522876735459 0 -48.52287673545898293 179.599720456223079643
	// which failed with Visual Studio 10 (Release and Debug)
	if cbet1 < -sbet1 {
		if cbet2 == cbet1 {
			if sbet2 < 0 {
				sbet2 = sbet1
			} else {
				sbet2 = -sbet1
			}
		}
	} else {
		if math.Abs(sbet2) == -sbet1 {
			cbet2 = cbet1
		}
	}

	lam12 := lon12
	slam12, clam12 := math.Sincos(lam12) // lon12 == 90 isn't interesting

	var sig12, calp1, salp1, calp2, salp2, omg12 float64
	// index zero elements of these arrays are unused
	var (
		C1a [_nC1 + 1]float64
		C2a [_nC2 + 1]float64
		C3a [_nC3]float64
	)

	meridian := lat1 == -math.Pi/2 || slam12 == 0.0

	if meridian {

		// Endpoints are on a single full meridian, so the geodesic might lie on
		// a meridian.

		calp1, salp2 = clam12, slam12 // Head to the target longitude
		calp2, salp2 = 1, 0           // At the target we're heading north

		// tan(bet) = tan(sig) * cos(alp)
		ssig1, csig1 := sbet1, calp1*cbet1
		ssig2, csig2 := sbet2, calp2*cbet2

		// sig12 = sig2 - sig1
		sig12 = math.Atan2(max(csig1*ssig2-ssig1*csig2, 0), csig1*csig2+ssig1*ssig2)

		s12x, m12x, _ = lengths(_n, sig12, ssig1, csig1, ssig2, csig2, cbet1, cbet2, C1a[:], C2a[:])

		// Add the check for sig12 since zero length geodesics might yield m12 < 0.  Test case was
		//
		//    echo 20.001 0 20.001 0 | Geod -i
		//
		// In fact, we will have sig12 > pi/2 for meridional geodesic which is
		// not a shortest path.
		if sig12 < 1 || m12x >= 0 {
			m12x *= _a
			s12x *= _b
		} else {
			// m12 < 0, i.e., prolate and too close to anti-podal
			meridian = false
		}

	}

	if !meridian && sbet1 == 0 && (_f <= 0 || lam12 <= math.Pi-_f*math.Pi) {

		// Geodesic runs along equator
		calp1, salp1, calp2, salp2 = 0, 1, 0, 1
		s12x = _a * lam12
		m12x = _b * math.Sin(lam12/_f1)
		omg12 = lam12 / _f1
		sig12 = omg12

	} else if !meridian {

		// Now point1 and point2 belong within a hemisphere bounded by a
		// meridian and geodesic is neither meridional or equatorial.

		// Figure a starting point for Newton's method
		sig12, salp1, calp1, salp2, calp2 = inverseStart(sbet1, cbet1, sbet2, cbet2, lam12, salp2, calp2, C1a[:], C2a[:])

		if sig12 >= 0 {

			// Short lines (InverseStart sets salp2, calp2)
			w1 := math.Sqrt(1 - _e2*cbet1*cbet1)
			s12x = sig12 * _a * w1
			m12x = w1 * w1 * _a / _f1 * math.Sin(sig12*_f1/w1)
			omg12 = lam12 / w1

		} else {

			// Newton's method
			var ssig1, csig1, ssig2, csig2, eps, ov float64
			numit := 0
			for trip := 0; numit < _maxit; numit++ {
				var v, dv float64

				v, salp2, calp2, sig12, ssig1, csig1, ssig2, csig2, eps, omg12, dv = 
					lambda12(sbet1, cbet1, sbet2, cbet2, salp1, calp1, trip < 1, C1a[:], C2a[:], C3a[:])
				v -= lam12

				if !(math.Abs(v) > _tiny) || !(trip < 1) {
					if !(math.Abs(v) <= max(_tol1, ov)) {
						numit = _maxit
					}
					break
				}

				dalp1 := -v / dv

				sdalp1, cdalp1 := math.Sincos(dalp1)
				nsalp1 := salp1*cdalp1 + calp1*sdalp1
				calp1 = calp1*cdalp1 - salp1*sdalp1
				salp1 = max(0, nsalp1)
				salp1, calp1 = sinCosNorm(salp1, calp1)

				if !(math.Abs(v) >= _tol1 && v*v >= ov*_tol0) {
					trip++
				}
				ov = math.Abs(v)
			}

			if numit >= _maxit {
				return math.NaN(), math.NaN(), math.NaN() // Signal failure.
			}

			s12x, m12x, _ = lengths(eps, sig12, ssig1, csig1, ssig2, csig2, cbet1, cbet2, C1a[:], C2a[:])

			m12x *= _a
			s12x *= _b
			omg12 = lam12 - omg12
		}
	}

	s12 = 0 + s12x // Convert -0 to 0

	// Convert calp, salp to azimuth accounting for lonsign, swapp, latsign.
	if swapp < 0 {
		salp1, salp2 = salp2, salp1
		calp1, calp2 = calp2, calp1
	}

	salp1 *= swapp * lonsign; calp1 *= swapp * latsign;
	salp2 *= swapp * lonsign; calp2 *= swapp * latsign;

	// minus signs give range [-180, 180). 0- converts -0 to +0.
	azi1 = 0 - math.Atan2(-salp1, calp1)
	azi2 = 0 - math.Atan2(salp2, -calp2) // make it point backwards

	return
}

// Return m12a = (reduced length)/_a; also calculate s12b = distance/_b,
// and m0 = coefficient of secular term in expression for reduced length.
func lengths(eps, sig12, ssig1, csig1, ssig2, csig2, cbet1, cbet2 float64, C1a, C2a []float64) (s12b, m12a, m0 float64) {

	c1f(eps, C1a)
	c2f(eps, C2a)

	A1m1 := a1m1f(eps)

	AB1 := (1 + A1m1) * (sinSeries(ssig2, csig2, C1a, _nC1) - sinSeries(ssig1, csig1, C1a, _nC1))

	A2m1 := a2m1f(eps)

	AB2 := (1 + A2m1) * (sinSeries(ssig2, csig2, C2a, _nC2) - sinSeries(ssig1, csig1, C2a, _nC2))

	cbet1sq, cbet2sq := cbet1*cbet1, cbet2*cbet2

	w1, w2 := math.Sqrt(1-_e2*cbet1sq), math.Sqrt(1-_e2*cbet2sq)

	// Make sure it's OK to have repeated dummy arguments
	m0 = A1m1 - A2m1
	J12 := m0*sig12 + (AB1 - AB2)

	// Missing a factor of _a.
	// Add parens around (csig1 * ssig2) and (ssig1 * csig2) to ensure accurate
	// cancellation in the case of coincident points.
	m12a = (w2*(csig1*ssig2) - w1*(ssig1*csig2)) - _f1*csig1*csig2*J12

	// Missing a factor of _b
	s12b = (1+A1m1)*sig12 + AB1

	return
}

// Return a starting point for Newton's method in salp1 and calp1 (function
// value is -1).  If Newton's method doesn't need to be used, return also
// salp2 and calp2 and function value is sig12.
func inverseStart(sbet1, cbet1, sbet2, cbet2, lam12, _salp2, _calp2 float64, C1a, C2a []float64) (sig12, salp1, calp1, salp2, calp2 float64) {

	sig12 = -1.
	salp2, calp2 = _salp2, _calp2
	// bet12 = bet2 - bet1 in [0, pi); bet12a = bet2 + bet1 in (-pi, 0]
	sbet12 := sbet2*cbet1 - cbet2*sbet1
	cbet12 := cbet2*cbet1 + sbet2*sbet1
	sbet12a := sbet2*cbet1 + cbet2*sbet1

	shortline := cbet12 >= 0 && sbet12 < 0.5 && lam12 <= math.Pi/6

	omg12 := lam12
	if shortline {
		omg12 = lam12 / math.Sqrt(1-_e2*cbet1*cbet1)
	}
	somg12, comg12 := math.Sincos(omg12)

	salp1 = cbet2 * somg12
	if comg12 >= 0 {
		calp1 = sbet12 + cbet2*sbet1*somg12*somg12/(1.+comg12)
	} else {
		calp1 = sbet12a - cbet2*sbet1*somg12*somg12/(1.-comg12)
	}

	ssig12 := math.Hypot(salp1, calp1)
	csig12 := sbet1*sbet2 + cbet1*cbet2*comg12

	if shortline && ssig12 < _etol2 {
		// really short lines
		salp2 = cbet1 * somg12
		calp2 = sbet12 - cbet1*sbet2*somg12*somg12/(1+comg12)
		salp2, calp2 = sinCosNorm(salp2, calp2)
		// Set return value
		sig12 = math.Atan2(ssig12, csig12)
	} else if csig12 >= 0 || ssig12 >= 3*math.Abs(_f)*math.Pi*cbet1*cbet1 {
		// Nothing to do, zeroth order spherical approximation is OK
	} else {
		// Scale lam12 and bet2 to x, y coordinate system where antipodal point
		// is at origin and singular point is at y = 0, x = -1.
		var x, y, lamscale, betscale float64
		if _f >= 0 { // In fact f == 0 does not get here
			// x = dlong, y = dlat
			k2 := sbet1 * sbet1 * _ep2
			eps := k2 / (2*(1+math.Sqrt(1+k2)) + k2)
			lamscale = _f * cbet1 * a3f(eps) * math.Pi
			betscale = lamscale * cbet1

			x = (lam12 - math.Pi) / lamscale
			y = sbet12a / betscale
		} else { // _f < 0
			// x = dlat, y = dlong
			cbet12a := cbet2*cbet1 - sbet2*sbet1
			bet12a := math.Atan2(sbet12a, cbet12a)

			// In the case of lon12 = 180, this repeats a calculation made in
			// Inverse.
			_, m12a, m0 := lengths(_n, math.Pi+bet12a, sbet1, -cbet1, sbet2, cbet2, cbet1, cbet2, C1a, C2a)

			x = -1 + m12a/(_f1*cbet1*cbet2*m0*math.Pi)
			if x < -0.01 {
				betscale = sbet12a / x
			} else {
				betscale = -_f * cbet1 * cbet1 * math.Pi
			}
			lamscale = betscale / cbet1
			y = (lam12 - math.Pi) / lamscale
		}

		if y > -_tol1 && x > -1-_xthresh {
			// strip near cut
			if _f >= 0 {
				salp1 = min(1, -x)
				calp1 = -math.Sqrt(1 - salp1*salp1)
			} else {
				if x > -_tol1 {
					calp1 = max(0, x)
				} else {
					calp1 = max(-1, x)
				}
				salp1 = math.Sqrt(1 - calp1*calp1)
			}
		} else {
			k := astroid(x, y)

			omg12a := lamscale
			if _f >= 0 {
				omg12a *= -x * k / (1 + k)
			} else {
				omg12a *= -y * (1 + k) / k
			}

			somg12, comg12 := math.Sincos(omg12a)
			comg12 = -comg12

			// Update spherical estimate of alp1 using omg12 instead of lam12
			salp1 = cbet2 * somg12
			calp1 = sbet12a - cbet2*sbet1*somg12*somg12/(1-comg12)
		}
	}

	salp1, calp1 = sinCosNorm(salp1, calp1)
	return
}

func lambda12(sbet1, cbet1, sbet2, cbet2, salp1, calp1 float64, diffp bool, C1a, C2a, C3a []float64) (lam12, salp2, calp2, sig12, ssig1, csig1, ssig2, csig2, eps, domg12, dlam12 float64) {

	// Break degeneracy of equatorial line.  This case has already been handled.
	if sbet1 == 0 && calp1 == 0 {
		calp1 = -_tiny
	}

	// sin(alp1) * cos(bet1) = sin(alp0)
	salp0 := salp1 * cbet1
	calp0 := math.Hypot(calp1, salp1*sbet1) // calp0 > 0

	var somg1, comg1, somg2, comg2, omg12 float64
	// tan(bet1) = tan(sig1) * cos(alp1)
	// tan(omg1) = sin(alp0) * tan(sig1) = tan(omg1)=tan(alp1)*sin(bet1)
	ssig1 = sbet1
	somg1, comg1 = salp0*sbet1, calp1*cbet1
	csig1 = comg1
	ssig1, csig1 = sinCosNorm(ssig1, csig1)
	// SinCosNorm(somg1, comg1); -- don't need to normalize!

	// Enforce symmetries in the case abs(bet2) = -bet1.  Need to be careful
	// about this case, since this can yield singularities in the Newton
	// iteration.
	// sin(alp2) * cos(bet2) = sin(alp0)
	if cbet2 != cbet1 {
		salp2 = salp0 / cbet2
	} else {
		salp2 = salp1
	}

	// calp2 = sqrt(1 - sq(salp2))
	//       = sqrt(sq(calp0) - sq(sbet2)) / cbet2
	// and subst for calp0 and rearrange to give (choose positive sqrt
	// to give alp2 in [0, pi/2]).
	if cbet2 != cbet1 || math.Abs(sbet2) != -sbet1 {
		var zz float64
		if cbet1 < -sbet1 {
			zz = (cbet2 - cbet1) * (cbet1 + cbet2)
		} else {
			zz = (sbet1 - sbet2) * (sbet1 + sbet2)
		}
		calp2 = math.Sqrt((calp1*cbet1)*(calp1*cbet1)+zz) / cbet2

	} else {
		calp2 = math.Abs(calp1)
	}

	// tan(bet2) = tan(sig2) * cos(alp2)
	// tan(omg2) = sin(alp0) * tan(sig2).
	ssig2 = sbet2
	somg2 = salp0 * sbet2
	comg2 = calp2 * cbet2
	csig2 = comg2
	ssig2, csig2 = sinCosNorm(ssig2, csig2)
	// SinCosNorm(somg2, comg2); -- don't need to normalize!

	// sig12 = sig2 - sig1, limit to [0, pi]
	sig12 = math.Atan2(max(csig1*ssig2-ssig1*csig2, 0), csig1*csig2+ssig1*ssig2)
	// omg12 = omg2 - omg1, limit to [0, pi]
	omg12 = math.Atan2(max(comg1*somg2-somg1*comg2, 0), comg1*comg2+somg1*somg2)
	var B312, h0 float64
	k2 := calp0 * calp0 * _ep2
	eps = k2 / (2*(1+math.Sqrt(1+k2)) + k2)
	c3f(eps, C3a)
	B312 = (sinSeries(ssig2, csig2, C3a, _nC3-1) - sinSeries(ssig1, csig1, C3a, _nC3-1))
	h0 = -_f * a3f(eps)
	domg12 = salp0 * h0 * (sig12 + B312)
	lam12 = omg12 + domg12
	if diffp {
		if calp2 == 0 {
			dlam12 = -2 * math.Sqrt(1-_e2*cbet1*cbet1) / sbet1
		} else {
			_, dlam12, _ = lengths(eps, sig12, ssig1, csig1, ssig2, csig2, cbet1, cbet2, C1a, C2a)
			dlam12 /= calp2 * cbet2
		}
	}

	return
}

// Solve k^4+2*k^3-(x^2+y^2-1)*k^2-2*y^2*k-y^2 = 0 for positive root k.
// This solution is adapted from Geocentric::Reverse.
func astroid(x, y float64) float64 {
	p, q := x*x, y*y
	r := (p + q - 1) / 6
	if q == 0 && r <= 0 {
		return 0
	}

	// Avoid possible division by zero when r = 0 by multiplying equations
	// for s and t by r^3 and r, resp.
	S := p * q / 4 // S = r^3 * s
	r2 := r * r
	r3 := r * r2
	// The discrimant of the quadratic equation for T3.  This is zero on
	// the evolute curve p^(1/3)+q^(1/3) = 1
	disc := S * (S + 2*r3)
	u := r
	if disc >= 0 {
		T3 := S + r3
		// Pick the sign on the sqrt to maximize abs(T3).  This minimizes loss
		// of precision due to cancellation.  The result is unchanged because
		// of the way the T is used in definition of u.
		if T3 < 0 {
			T3 += -math.Sqrt(disc)
		} else {
			T3 += math.Sqrt(disc) // T3 = (r * t)^3
		}
		// N.B. cbrt always returns the real root.  cbrt(-8) = -2.
		T := math.Cbrt(T3) // T = r * t
		// T can be zero; but then r2 / T -> 0.
		if T != 0 {
			u += T + r2/T
		}
	} else {
		// T is complex, but the way u is defined the result is real.
		ang := math.Atan2(math.Sqrt(-disc), -(S + r3))
		// There are three possible cube roots.  We choose the root which
		// avoids cancellation.  Note that disc < 0 implies that r < 0.
		u += 2 * r * math.Cos(ang/3)
	}

	v := math.Sqrt(u*u + q) // guaranteed positive
	// Avoid loss of accuracy when u < 0.
	var uv float64
	if u < 0 {
		uv = q / (v - u)
	} else {
		uv = u + v // u+v, guaranteed positive
	}
	w := (uv - q) / (2 * v) // positive?
	// Rearrange expression for k to avoid loss of accuracy due to
	// subtraction.  Division by 0 not possible because uv > 0, w >= 0.
	return uv / (math.Sqrt(uv+w*w) + w) // guaranteed positive
}
