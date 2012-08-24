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

// This package provides forward and inverse geodesic calculations
// on the WGS84 ellipsoid.
//
// The calculations are borrowed from the GeographicLib C++ library by
// Charles Karney.  Its Geoid class is replaced by hardcoded WGS84
// with the sperical harmonics computed to 8'th order.  The forward
// function is lifted from the C++ code for GeodesicLine(lat1, lon1,
// azi1..).GenPosition(s12...) and it's dependents, the inverse
// function is Geodesic::GenInverse.
//
// The documentation for the Original C++ classes can be found at
//     http://geographiclib.sourceforge.net/html/annotated.html
//
package wgs84

import "math"

const (
	WGS84_a = 6378137.             // Equatorial radius in meters
	WGS84_f = 1. / 298.25722210088 // Flattening of the ellipsoid
)

// Evaluate
//  sum(c[i] * sin( 2*i    * x), i, 1, n) 
//  sum(c[i] * cos((2*i+1) * x), i, 0, n-1)
// using Clenshaw summation.  N.B. c[0] is unused for sin series
func sinSeries(sinx, cosx float64, c []float64, n int) float64 {
	cp := n + 1
	ar := 2 * (cosx - sinx) * (cosx + sinx) // 2 * cos(2 * x)
	var y0, y1 float64
	if n&1 != 0 {
		cp--
		y0 = c[cp]
	}
	for n /= 2; n > 0; n-- {
		cp--
		y1 = ar*y0 - y1 + c[cp]
		cp--
		y0 = ar*y1 - y0 + c[cp]
	}
	return 2 * sinx * cosx * y0
}

func cosSeries(sinx, cosx float64, c []float64, n int) float64 {
	cp := n
	ar := 2 * (cosx - sinx) * (cosx + sinx) // 2 * cos(2 * x)
	var y0, y1 float64
	if n&1 != 0 {
		cp--
		y0 = c[cp]
	}
	for n /= 2; n > 0; n-- {
		cp--
		y1 = ar*y0 - y1 + c[cp]
		cp--
		y0 = ar*y1 - y0 + c[cp]
	}
	return cosx * (y0 - y1)
}

func min(x, y float64) float64 {
	if x > y {
		return y
	}
	return x
}

func max(x, y float64) float64 {
	if x < y {
		return y
	}
	return x
}

// put rad in [-pi, +pi)
func angNormalize(rad float64) float64 {
	for rad < -math.Pi {
		rad += 2 * math.Pi
	}
	for rad >= math.Pi {
		rad -= 2 * math.Pi
	}
	return rad
}

func angRound(x float64) float64 {
	// Use this to avoid having to deal with near singular
	// cases when x is non-zero but tiny (e.g., 1.0e-200).
	x *= (180. / math.Pi) // original worked on degrees
	var z float64 = 0.0625
	y := math.Abs(x)
	if y < z {
		y = z - y
		y = z - y
	}
	y *= (math.Pi / 180.)
	if x < -0 {
		return -y
	}
	return y
}

func sinCosNorm(s, c float64) (sn, cn float64) { r := math.Hypot(s, c); return s / r, c / r }

func a1m1f(eps float64) float64 {
	eps2 := eps * eps
	t := eps2 * (eps2*(eps2*(25*eps2+64)+256) + 4096) / 16384.
	return (t + eps) / (1 - eps)
}

func a2m1f(eps float64) float64 {
	eps2 := eps * eps
	t := eps2 * (eps2*(eps2*(1225*eps2+1600)+2304) + 4096) / 16384.
	return t*(1-eps) - eps
}

func a3f(eps float64) float64 {
	var v float64
	for i := _nA3x - 1; i >= 0; i-- {
		v = eps*v + _A3x[i]
	}
	return v
}

func c1f(eps float64, c []float64) {
	eps2 := eps * eps
	d := eps
	c[1] = d * (eps2*(eps2*(19*eps2-64)+384) - 1024) / 2048.
	d *= eps
	c[2] = d * (eps2*(eps2*(7*eps2-18)+128) - 256) / 4096.
	d *= eps
	c[3] = d * ((72-9*eps2)*eps2 - 128) / 6144.
	d *= eps
	c[4] = d * ((96-11*eps2)*eps2 - 160) / 16384.
	d *= eps
	c[5] = d * (35*eps2 - 56) / 10240.
	d *= eps
	c[6] = d * (9*eps2 - 14) / 4096.
	d *= eps
	c[7] = -33 * d / 14336.
	d *= eps
	c[8] = -429 * d / 262144.
}

func c1pf(eps float64, c []float64) {
	eps2 := eps * eps
	d := eps
	c[1] = d * (eps2*((9840-4879*eps2)*eps2-20736) + 36864) / 73728.
	d *= eps
	c[2] = d * (eps2*((120150-86171*eps2)*eps2-142080) + 115200) / 368640.
	d *= eps
	c[3] = d * (eps2*(8703*eps2-7200) + 3712) / 12288.
	d *= eps
	c[4] = d * (eps2*(1082857*eps2-688608) + 258720) / 737280.
	d *= eps
	c[5] = d * (41604 - 141115*eps2) / 92160.
	d *= eps
	c[6] = d * (533134 - 2200311*eps2) / 860160.
	d *= eps
	c[7] = 459485 * d / 516096.
	d *= eps
	c[8] = 109167851 * d / 82575360.
}

// The coefficients C2[l] in the Fourier expansion of B2
func c2f(eps float64, c []float64) {
	eps2 := eps * eps
	d := eps
	c[1] = d * (eps2*(eps2*(41*eps2+64)+128) + 1024) / 2048.
	d *= eps
	c[2] = d * (eps2*(eps2*(47*eps2+70)+128) + 768) / 4096.
	d *= eps
	c[3] = d * (eps2*(69*eps2+120) + 640) / 6144.
	d *= eps
	c[4] = d * (eps2*(133*eps2+224) + 1120) / 16384.
	d *= eps
	c[5] = d * (105*eps2 + 504) / 10240.
	d *= eps
	c[6] = d * (33*eps2 + 154) / 4096.
	d *= eps
	c[7] = 429 * d / 14336.
	d *= eps
	c[8] = 6435 * d / 262144.
}

func c3f(eps float64, c []float64) {
	for j, k := _nC3x, _nC3-1; k > 0; k-- {
		var t float64
		for i := _nC3 - k; i > 0; i-- {
			j--
			t = eps*t + _C3x[j]
		}
		c[k] = t
	}

	mult := 1.
	for k := 1; k < _nC3; k++ {
		mult *= eps
		c[k] *= mult
	}
}

var (
	_tiny    = math.Sqrt((1<<52)*math.SmallestNonzeroFloat64) // sqrt(smallest normalized number)
	_tol0    = 1.0 / (1<<52)  // epsilon for a 52 bit mantissa
	_tol1    = 200 * _tol0
	_tol2    = math.Sqrt(_tol0)
	_xthresh = 1000 * _tol2
)

var _a, _f, _f1, _e2, _ep2, _n, _b, _c2, _etol2 float64

const (
	_GEOD_ORD = 8
	_nA1      = _GEOD_ORD
	_nC1      = _GEOD_ORD
	_nC1p     = _GEOD_ORD
	_nA2      = _GEOD_ORD
	_nC2      = _GEOD_ORD
	_nA3      = _GEOD_ORD
	_nA3x     = _nA3
	_nC3      = _GEOD_ORD
	_nC3x     = (_nC3 * (_nC3 - 1)) / 2
	_nC4      = _GEOD_ORD
	_nC4x     = (_nC4 * (_nC4 + 1)) / 2
	_maxit    = 50
)

var (
	_A3x [_nA3x]float64
	_C3x [_nC3x]float64
	_C4x [_nC4x]float64
)

func init() {
	_a = WGS84_a
	_f = WGS84_f

	_f1 = 1 - _f
	_e2 = _f * (2 - _f)
	_ep2 = _e2 / (_f1 * _f1)
	_n = _f / (2 - _f)
	_b = _a * _f1

	_c2 = _b * _b
	switch {
	case _e2 > 0:
		_c2 *= math.Atanh(math.Sqrt(_e2)) / math.Sqrt(math.Abs(_e2))
	case _e2 < 0:
		_c2 *= math.Atan(math.Sqrt(-_e2)) / math.Sqrt(math.Abs(_e2))
	}
	_c2 += _a * _a
	_c2 /= 2

	if math.Abs(_e2) < 0.01 {
		_etol2 = _tol2 / 0.1
	} else {
		_etol2 = _tol2 / math.Sqrt(math.Abs(_e2))
	}

	_A3x[0] = 1.
	_A3x[1] = (_n - 1) / 2.
	_A3x[2] = (_n*(3*_n-1) - 2) / 8.
	_A3x[3] = (_n*(_n*(5*_n-1)-3) - 1) / 16.
	_A3x[4] = (_n*((-5*_n-20)*_n-4) - 6) / 128.
	_A3x[5] = ((-5*_n-10)*_n - 6) / 256.
	_A3x[6] = (-15*_n - 20) / 1024.
	_A3x[7] = -25. / 2048.

	_C3x[0] = (1 - _n) / 4.
	_C3x[1] = (1 - _n*_n) / 8.
	_C3x[2] = (_n*((-5*_n-1)*_n+3) + 3) / 64.
	_C3x[3] = (_n*((2-2*_n)*_n+2) + 5) / 128.
	_C3x[4] = (_n*(3*_n+11) + 12) / 512.
	_C3x[5] = (10*_n + 21) / 1024.
	_C3x[6] = 243. / 16384.
	_C3x[7] = ((_n-3)*_n + 2) / 32.
	_C3x[8] = (_n*(_n*(2*_n-3)-2) + 3) / 64.
	_C3x[9] = (_n*((-6*_n-9)*_n+2) + 6) / 256.
	_C3x[10] = ((1-2*_n)*_n + 5) / 256.
	_C3x[11] = (69*_n + 108) / 8192.
	_C3x[12] = 187. / 16384.
	_C3x[13] = (_n*((5-_n)*_n-9) + 5) / 192.
	_C3x[14] = (_n*(_n*(10*_n-6)-10) + 9) / 384.
	_C3x[15] = ((-77*_n-8)*_n + 42) / 3072.
	_C3x[16] = (12 - _n) / 1024.
	_C3x[17] = 139. / 16384.
	_C3x[18] = (_n*((20-7*_n)*_n-28) + 14) / 1024.
	_C3x[19] = ((-7*_n-40)*_n + 28) / 2048.
	_C3x[20] = (72 - 43*_n) / 8192.
	_C3x[21] = 127 / 16384.
	_C3x[22] = (_n*(75*_n-90) + 42) / 5120.
	_C3x[23] = (9 - 15*_n) / 1024.
	_C3x[24] = 99. / 16384.
	_C3x[25] = (44 - 99*_n) / 8192.
	_C3x[26] = 99. / 16384.
	_C3x[27] = 429. / 114688.

	_C4x[0] = (_ep2*(_ep2*(_ep2*(_ep2*(_ep2*((8704-7168*_ep2)*_ep2-10880)+14144)-19448)+29172)-51051) + 510510) / 765765.
	_C4x[1] = (_ep2*(_ep2*(_ep2*(_ep2*((8704-7168*_ep2)*_ep2-10880)+14144)-19448)+29172) - 51051) / 1021020.
	_C4x[2] = (_ep2*(_ep2*(_ep2*((2176-1792*_ep2)*_ep2-2720)+3536)-4862) + 7293) / 306306.
	_C4x[3] = (_ep2*(_ep2*((1088-896*_ep2)*_ep2-1360)+1768) - 2431) / 175032.
	_C4x[4] = (_ep2*((136-112*_ep2)*_ep2-170) + 221) / 24310.
	_C4x[5] = ((68-56*_ep2)*_ep2 - 85) / 13260.
	_C4x[6] = (17 - 14*_ep2) / 3570.
	_C4x[7] = -1. / 272.
	_C4x[8] = (_ep2*(_ep2*(_ep2*(_ep2*(_ep2*(7168*_ep2-8704)+10880)-14144)+19448)-29172) + 51051) / 9189180.
	_C4x[9] = (_ep2*(_ep2*(_ep2*(_ep2*(1792*_ep2-2176)+2720)-3536)+4862) - 7293) / 1837836.
	_C4x[10] = (_ep2*(_ep2*(_ep2*(896*_ep2-1088)+1360)-1768) + 2431) / 875160.
	_C4x[11] = (_ep2*(_ep2*(112*_ep2-136)+170) - 221) / 109395.
	_C4x[12] = (_ep2*(56*_ep2-68) + 85) / 55692.
	_C4x[13] = (14*_ep2 - 17) / 14280.
	_C4x[14] = 7. / 7344.
	_C4x[15] = (_ep2*(_ep2*(_ep2*((2176-1792*_ep2)*_ep2-2720)+3536)-4862) + 7293) / 15315300.
	_C4x[16] = (_ep2*(_ep2*((1088-896*_ep2)*_ep2-1360)+1768) - 2431) / 4375800.
	_C4x[17] = (_ep2*((136-112*_ep2)*_ep2-170) + 221) / 425425.
	_C4x[18] = ((68-56*_ep2)*_ep2 - 85) / 185640.
	_C4x[19] = (17 - 14*_ep2) / 42840.
	_C4x[20] = -7. / 20400.
	_C4x[21] = (_ep2*(_ep2*(_ep2*(896*_ep2-1088)+1360)-1768) + 2431) / 42882840.
	_C4x[22] = (_ep2*(_ep2*(112*_ep2-136)+170) - 221) / 2382380.
	_C4x[23] = (_ep2*(56*_ep2-68) + 85) / 779688.
	_C4x[24] = (14*_ep2 - 17) / 149940.
	_C4x[25] = 1. / 8976.
	_C4x[26] = (_ep2*((136-112*_ep2)*_ep2-170) + 221) / 27567540.
	_C4x[27] = ((68-56*_ep2)*_ep2 - 85) / 5012280.
	_C4x[28] = (17 - 14*_ep2) / 706860.
	_C4x[29] = -7. / 242352.
	_C4x[30] = (_ep2*(56*_ep2-68) + 85) / 67387320.
	_C4x[31] = (14*_ep2 - 17) / 5183640.
	_C4x[32] = 7. / 1283568.
	_C4x[33] = (17 - 14*_ep2) / 79639560.
	_C4x[34] = -1. / 1516944.
	_C4x[35] = 1. / 26254800.

}
