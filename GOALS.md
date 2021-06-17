To:

1. Report on github builds (taken from the notifications endpoint)
1. Allow for the re-running of broken builds (we'll store some internal state)
1. Maybe trigger builds for other job, where a workflow trigger exists (I'd quite like to do something sensible, like a 'reload' that can re-read all repos and figure out which repos have a workflow trigger)
