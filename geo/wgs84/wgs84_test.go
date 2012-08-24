// Copyright 2011 The Avalon Project Authors. All rights reserved.
// Use of this source code is governed by the Apache License 2.0
// that can be found in the LICENSE file.

//
// Test by comparing against an independent implementation of the inverse problem,
// published by Gerald I. Evenden.  I understand this is in the public domain.
//
package wgs84

import (
	"math"
	"testing"
)

const tol = 5E-10

func rad(deg float64) float64 { return deg * (math.Pi / 180.0) }
func deg(rad float64) float64 { return rad * (180.0 / math.Pi) }

func TestOne(t *testing.T) {
	lat1, lon1, azi1 := rad(33.), rad(-91.5), rad(23.361326677)
	dist := 1100896.2093
	lat2, lon2, azi2 := Forward(lat1, lon1, azi1, dist)

	s, faz, baz := inv_geodesic(lat1, lon1, lat2, lon2)

	if e := math.Abs(s - dist); !(e < tol*dist) {
		t.Errorf("bad dist %g, %g", s, dist)
	}
	if e := math.Abs(faz - azi1); !(e < tol) {
		t.Errorf("bad azi1 %g %g", deg(faz), deg(azi1))
	}
	if e := math.Abs(baz - azi2); !(e < tol) {
		t.Errorf("bad azi2 %g %g", deg(baz), deg(azi2))
	}
}

func TestTwo(t *testing.T) {
	lat1, lon1, azi1 := rad(33.), rad(-91.5), rad(23.361326677)
	dist := 1100896.2093
	lat2, lon2, azi2 := Forward(lat1, lon1, azi1, dist)
	s, faz, baz := Inverse(lat1, lon1, lat2, lon2)
	if e := math.Abs(s - dist); !(e < tol*dist) {
		t.Errorf("bad dist %g, %g", s, dist)
	}
	if e := math.Abs(faz - azi1); !(e < tol) {
		t.Errorf("bad azi1 %g %g", deg(faz), deg(azi1))
	}
	if e := math.Abs(baz - azi2); !(e < tol) {
		t.Errorf("bad azi2 %g %g", deg(baz), deg(azi2))
	}

}

var (
	lats = [...]float64{0, 30, 45, 60, 89,  33., 42.  } 
	lons = [...]float64{0, 15, 45, 60, -91.5, -86.25 } 
)

func TestForward(t *testing.T) {
	for _, lat1 := range lats {
		for _, lon1 := range lons {
			for _, azi1 := range lons {
			for s := 1000.; s < 10001; s += 1000 {
					lat2, lon2, azi2 := Forward(rad(lat1), rad(lon1), rad(azi1), s)
					rs, razi1, razi2 := inv_geodesic(rad(lat1), rad(lon1), lat2, lon2)

					if e := math.Abs(rs - s) / s; !(e < 1E-5) {
						t.Errorf("(%g,%g) -> (%g,%g) bad dist %g, %g", lat1, lon1, lat2, lon2, rs, s)
					} else if !(s < tol) {
						if e := math.Abs(rad(azi1) - razi1); !(e < tol) {
							t.Errorf("(%g,%g) -> (%g,%g) bad faz %g %g", lat1, lon1, lat2, lon2, azi1, deg(razi1))
						}
						if e := math.Abs(azi2 - razi2); !(e < tol) {
							t.Errorf("(%g,%g) -> (%g,%g) bad baz %g %g", lat1, lon1, lat2, lon2, deg(azi2), deg(razi2))
						}
					}



				}
			}
		}
	}
}

func TestInverse(t *testing.T) {
	for _, lat1 := range lats {
		for _, lon1 := range lons {
			for _, lat2 := range lats {
				for _, lon2 := range lons {
					if lat1 == lat2 && lon1 == lon2 {
						continue
					}

					s,  faz, baz := Inverse(rad(lat1), rad(lon1), rad(lat2), rad(lon2))
					rs, rfaz, rbaz := inv_geodesic(rad(lat1), rad(lon1), rad(lat2), rad(lon2))

					if e := math.Abs(rs - s) / s; !(e < 1E-5) {
						t.Errorf("(%g,%g) -> (%g,%g) bad dist %g, %g", lat1, lon1, lat2, lon2, rs, s)
					} else if !(s < tol) {
						if e := math.Abs(rfaz - faz); !(e < tol) {
							t.Errorf("(%g,%g) -> (%g,%g) bad faz %g %g", lat1, lon1, lat2, lon2, deg(rfaz), deg(faz))
						}
						if e := math.Abs(rbaz - baz); !(e < tol) {
							t.Errorf("(%g,%g) -> (%g,%g) bad baz %g %g", lat1, lon1, lat2, lon2, deg(rbaz), deg(baz))
						}
					}
				}
			}
		}
	}
}

/*

 Distance and bearing calculations on the  GRS80 / WGS84  (NAD83) or Clark 66 Ellipsoid.

 http://article.gmane.org/gmane.comp.gis.proj-4.devel/3478

 From: Gerald I. Evenden <geraldi.evenden <at> gmail.com>
 Subject: Re: Any access to geodetic[sic] functions ??
 Newsgroups: gmane.comp.gis.proj-4.devel
 Date: 2008-11-14 21:04:53 GMT (3 years, 2 weeks, 4 days, 22 hours and 14 minutes ago)
 On Friday 14 November 2008 3:18:23 pm Christopher Barker wrote:
 > Gerald I. Evenden wrote:
 > > On Thursday 13 November 2008 1:18:27 pm Christopher Barker wrote:
 ...
 > I'd like to see it.

 Ok, you asked for it.  The inverse problem:
*/

// Translation of NGS FORTRAN code for determination of true distance
// and respective forward and back azimuths between two points on the
// ellipsoid.  Good for any pair of points that are not antipodal.
//
//      INPUT
// 	phi1, lam1 -- latitude and longitude of first point in radians.
// 	phi2, lam2 -- latitude and longitude of second point in radians.
//
// 	OUTPUT
// 	s    -- distance between points normalized by major elliptical axis (i.e. a * s to get distance).
//  	Az12 -- azimuth from first point to second in radians clockwise	from North.
// 	Az12 -- azimuth from second point back to first point.

func inv_geodesic(phi1, lam1, phi2, lam2 float64) (s, faz, baz float64) {

	const (
		f   = WGS84_f
		a   = WGS84_a
		r   = 1. - f
		eps = 5E-14 // orig says 5E-14
	)

	tu1 := r * math.Tan(phi1)
	tu2 := r * math.Tan(phi2)
	cu1 := 1. / math.Sqrt(tu1*tu1+1.)
	su1 := cu1 * tu1
	cu2 := 1. / math.Sqrt(tu2*tu2+1.)
	ts := cu1 * cu2
	baz = ts * tu2
	faz = baz * tu1
	x := lam2 - lam1

	var c, d, e, y, sa, cx, cy, cz, sx, sy, c2a float64

	for it := 0; it < 30; it++ {
		sx = math.Sin(x)
		cx = math.Cos(x)
		tu1 = cu2 * sx
		tu2 = baz - su1*cu2*cx
		sy = math.Sqrt(tu1*tu1 + tu2*tu2)
		cy = ts*cx + faz
		y = math.Atan2(sy, cy)
		sa = ts * sx / sy
		c2a = -sa*sa + 1.
		cz = faz + faz

		if c2a > 0. {
			cz = -cz/c2a + cy
		}

		e = cz*cz*2. - 1.
		c = ((c2a*-3.+4.)*f + 4.) * c2a * f / 16.
		d = x
		x = ((e*cy*c+cz)*sy*c + y) * sa
		x = (1.-c)*x*f + lam2 - lam1

		if math.Abs(d-x) < eps {
			break
		}
	}

	faz = math.Atan2(tu1, tu2)
	baz = math.Atan2(cu1*sx, baz*cx-su1*cu2) + math.Pi
	if baz >= math.Pi {
		baz -= math.Pi * 2
	}
	x = math.Sqrt((1./r/r-1.)*c2a+1.) + 1.
	x = (x - 2.) / x
	c = (x*x/4. + 1.) / (1. - x)
	d = (x*.375*x - 1.) * x
	s = ((((sy*sy*4.-3.)*(1.-e-e)*cz*d/6.-e*cy)*d/4.+cz)*sy*d + y) * c * r
	s *= a
	return
}

/*
// Check on the Clark 66 ellipsoid.  set the constants f and a to these instead:
const (
	CLARK66_a = 6378206.4 
	CLARK66_f = 1 / 294.9786982138

	WGS84_a   = 6378137             // Equatorial radius in meters
	WGS84_f   = 1 / 298.25722210088 // Flattening of the ellipsoid
)

func main() {
	A12, A21, s := inv_geodesic(rad(33.), rad(-91.5), rad(42.), rad(-86.25))
//	fmt.Printf("expect fwd Az: %.8f, back Az: %.8f, distance: %.4f\n", 23.361326677, 206.568647963, 1100896.2093)
	fmt.Printf("expect fwd Az: %.8f, back Az: %.8f, distance: %.4f\n", 23.361326677, 206.568647963 - 360, 1100896.2093)
	fmt.Printf("got    fwd Az: %.8f, back Az: %.8f, distance: %.4f\n", deg(A12), deg(A21), s)
}
*/
