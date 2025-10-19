import { Component } from '@angular/core';
import { NgClass, NgFor, NgIf } from '@angular/common';
import { RouterLink, RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, RouterLink, NgFor, NgIf, NgClass],
  templateUrl: './app.component.html',
  styleUrl: './app.component.css'
})
export class AppComponent {
  readonly title = 'FreqShow';
  readonly navLinks = [
    { label: 'Home', path: '/' },
    { label: 'Artists', note: 'soon' },
    { label: 'Albums', note: 'soon' },
    { label: 'Genres', note: 'soon' },
    { label: 'Reviews', note: 'soon' },
  ];
}
