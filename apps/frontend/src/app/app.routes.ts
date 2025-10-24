import { Routes } from '@angular/router';
import { HomeComponent } from './pages/home/home.component';
import { ArtistDetailComponent } from './pages/artist-detail/artist-detail.component';
import { AlbumDetailComponent } from './pages/album-detail/album-detail.component';

export const routes: Routes = [
	{ path: '', component: HomeComponent },
	{ path: 'artists/:id', component: ArtistDetailComponent },
	{ path: 'albums/:id', component: AlbumDetailComponent },
	{ path: '**', redirectTo: '' },
];
