import { Component } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { AppComponent } from './app.component';
import { SearchService } from './services/search.service';
import { ApiStatusService } from './services/api-status.service';

@Component({
  standalone: true,
  template: ''
})
class DummyComponent {}

describe('AppComponent', () => {
  let searchServiceSpy: jasmine.SpyObj<SearchService>;
  let apiStatusServiceStub: Pick<ApiStatusService, 'init' | 'status$'>;

  beforeEach(async () => {
    searchServiceSpy = jasmine.createSpyObj<SearchService>('SearchService', ['requestSearchReset']);
    apiStatusServiceStub = {
      init: jasmine.createSpy('init'),
      status$: of('ready')
    };

    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideRouter([
          { path: '', component: DummyComponent },
          { path: 'artists/:id', component: DummyComponent },
        ]),
        { provide: SearchService, useValue: searchServiceSpy },
        { provide: ApiStatusService, useValue: apiStatusServiceStub }
      ],
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it(`should have the 'FreqShow' title`, () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;
    expect(app.title).toEqual('FreqShow');
  });

  it('should render the brand in the header', () => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('header a')?.textContent).toContain('FreqShow');
  });

  it('should clear search state when the home link is used', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;

    app.onHomeClick();

    expect(searchServiceSpy.requestSearchReset).toHaveBeenCalled();
  });

  it('should clear search state after navigating away from home', async () => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const router = TestBed.inject(Router);
    await router.navigateByUrl('/artists/123');
    await fixture.whenStable();

    expect(searchServiceSpy.requestSearchReset).toHaveBeenCalled();
  });
});
