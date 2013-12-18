## Introduction
This is an [IVR](http://en.wikipedia.org/wiki/Interactive_voice_response) application base on  [FreeSWITCH](http://www.freeswitch.org/) which implements by [Go Language](http://golang.org/).

FreeSWITCH's EventSocket supported both inbound and outbound mode,and this application uses outbound mode, so FreeSWITCH must connect to it actively.

## Usage

You can run it on Linux OS directly as following command :

		cd src
		chmod 775 *
		./src
		
Notice : When you run it you must edit ivr.xml file first,In this file you can edit your call flow with *Prompt*,*Grammar* and *Node*.

On other Platform you must recompile and then run it.	


