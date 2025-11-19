package droplet

//go:generate mockgen -destination=./mocks.go -package droplet github.com/digitalocean/godo  DropletsService,DropletActionsService,SizesService,ImagesService
