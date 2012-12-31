#!/bin/bash

BASE_URL="http://www.unidata.ucar.edu/software/netcdf/examples/"
FILES="
04091217_ruc.nc
19981111_0045.nc
ECMWF_ERA-40_subset.nc
GLASS.nc
GOTEX.C130_N130AR.LRT.RF06.PNI.nc
HRDL_iop12_19991027024421.nc
IMAGE0002.nc
WMI_Lear.nc
cami_0000-09-01_64x128_L26_c030918.nc
madis-hydro.nc
madis-maritime.nc
madis-mesonet.nc
madis-metar.nc
madis-profiler.nc
madis-raob.nc
madis-sao.nc
ncswp_SPOL_RHI_.nc
rhum.2003.nc
sgpsondewnpnC1.nc
slim_100897_198.nc
smith_sandwell_topo_v8_2.nc
sresa1b_ncar_ccsm3_0_run1_200001.nc
tos_O1_2001-2002.nc
wrfout_v2_Lambert.nc
"

if `which wget > /dev/null 2>&1` ; then
    GET="wget"
elif `which curl > /dev/null 2>&1` ; then
    GET="curl -O"
else
    echo "No wget or curl. You may manually fetch the following files and place them in this directory."
    GET=echo
fi

for f in $FILES; do
    if [ -r $f ]; then
	echo "already have $f, skipping"
    else
	echo $GET ${BASE_URL}$f
    fi
done
