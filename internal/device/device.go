/*
** Copyright (C) 2026 Key9, Inc <k9.io>
** Copyright (C) 2026 Champ Clark III <cclark@k9.io>
**
** This file is part of the HighVolt JSON analysis engine
**
** This program is free software: you can redistribute it and/or modify
** it under the terms of the GNU Affero General Public License as published by
** the Free Software Foundation, either version 3 of the License, or
** (at your option) any later version.
**
** This program is distributed in the hope that it will be useful
** but WITHOUT ANY WARRANTY; without even the implied warranty of
** MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
** GNU Affero General Public License for more details.
**
** You should have received a copy of the GNU Affero General Public License
** along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package device

import ( 

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"

	"github.com/tidwall/sjson"

)

type Device_Struct struct {
	OS      OS_Struct
	CPU     CPU_Struct
	Memory  Memory_Struct
	Network []Network_Struct
}

type OS_Struct struct {
	Hostname string
	OS       string
	Platform string
	Version  string
	Kernel   string
}

type CPU_Struct struct {
	Model  string
	Cores  int32
	Vendor string
}

type Memory_Struct struct {
	Total_Mem uint64
}

type Network_Struct struct {
	Interface_Name    string
	Interface_Address string
}

var Device_Info Device_Struct

/***********************************************************************/
/* Get_Device_Info - Get basic device information.  This can be useful */
/* in tracking where documents came from                               */
/***********************************************************************/

func Get_Device_Info() (error) {

//	var Device_Info Device_Struct

	/* Get OS level information (hostname, OS, etc) */
	
	hInfo, err := host.Info()

	if err != nil {
		return err
	}

	Device_Info.OS.Hostname = hInfo.Hostname
	Device_Info.OS.OS = hInfo.OS
	Device_Info.OS.Platform = hInfo.Platform
	Device_Info.OS.Version = hInfo.PlatformVersion 
	Device_Info.OS.Kernel = hInfo.KernelVersion

	/* Get CPU Information (model, cores, etc) */

	cInfo, err := cpu.Info()

	if err != nil {
		return err
	}

	Device_Info.CPU.Model = cInfo[0].ModelName
	Device_Info.CPU.Cores = cInfo[0].Cores
	Device_Info.CPU.Vendor = cInfo[0].VendorID

	/* Get RAM information (64 gb, etc)  */

	mInfo, err := mem.VirtualMemory()

	if err != nil {
		return err
	}

	Device_Info.Memory.Total_Mem = mInfo.Total/1024/1024/1024

	/* Network information */

	var TMP_NET Network_Struct

	interfaces, err := net.Interfaces()

	if err != nil {
		return err
	}

	for _, interf := range interfaces {

		for _, addr := range interf.Addrs {

			TMP_NET.Interface_Name = interf.Name
			TMP_NET.Interface_Address = addr.Addr

			Device_Info.Network = append(Device_Info.Network, TMP_NET)
		
		}
	}

return nil
}

/***************************************************************************/
/* Device_Info_JSON - Populate the highvolt JSON with device information.  */
/* Useful for figuring out "where" the data originally comes from.         */
/***************************************************************************/

func Device_Info_JSON( highvolt_json string ) string { 

	/* Populate OS Data */

	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.os.hostname", Device_Info.OS.Hostname)
	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.os.os", Device_Info.OS.OS)
	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.os.platform", Device_Info.OS.Platform)
	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.os.version", Device_Info.OS.Version)
	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.os.kernel", Device_Info.OS.Kernel)

	/* CPU */

	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.cpu.model", Device_Info.CPU.Model)
	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.cpu.cores", Device_Info.CPU.Cores)
	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.cpu.vendor", Device_Info.CPU.Vendor)

	/* Memory */

	highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.memory.total_mem", Device_Info.Memory.Total_Mem)

	/* IP Addresses */

	for _, i := range Device_Info.Network {

		highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.network.interface", i.Interface_Name)
		highvolt_json, _ = sjson.Set(highvolt_json, "data_origin.network.address", i.Interface_Address)

	}

	return highvolt_json

}
